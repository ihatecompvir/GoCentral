package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/natefinch/lumberjack"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	database "rb3server/database"
	"rb3server/restapi"
	"rb3server/servers"
)

func main() {

	envFile := flag.String("env", ".env", "specify the .env file to load")
	flag.Parse()

	err := godotenv.Load(*envFile)
	if err != nil {
		log.Println("Error loading .env file, using environment variables instead")
	}

	// if the user has set a log path, log there, otherwise log to stdout
	logPath := os.Getenv("LOGPATH")

	if logPath != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    10,   // Max size in MB before rotation
			MaxBackups: 3,    // Max number of old log files to retain
			MaxAge:     28,   // Max number of days to retain old log files
			Compress:   true, // Compress/zip old log files
		})
	}

	ticketVerifierEndpoint := os.Getenv("TICKETVERIFIERENDPOINT")

	if ticketVerifierEndpoint == "" {
		log.Println("Ticket verification is disabled, GoCentral will have no real authentication! Please do not use this server in a production environment.")
	}

	uri := os.Getenv("MONGOCONNECTIONSTRING")

	if uri == "" {
		log.Fatalln("GoCentral relies on MongoDB. You must set a MongoDB connection string to use GoCentral")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {
		log.Fatalln("Could not connect to MongoDB: ", err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatalln("Could not connect to MongoDB: ", err)
		}
	}()

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalln("Could not ping MongoDB: ", err)
	}

	log.Println("Successfully established connection to MongoDB")

	database.GocentralDatabase = client.Database("gocentral")

	configCollection := database.GocentralDatabase.Collection("config")

	// get config from DB
	err = configCollection.FindOne(nil, bson.M{}).Decode(&servers.Config)
	if err != nil {
		log.Println("Could not get config from MongoDB database, creating default config: ", err)
		_, err = configCollection.InsertOne(nil, bson.D{
			{Key: "last_pid", Value: 500},
			{Key: "last_band_id", Value: 0},
			{Key: "last_character_id", Value: 0},
			{Key: "last_setlist_id", Value: 0},
			{Key: "profanity_list", Value: []string{}},
			{Key: "battle_limit", Value: 5},
			{Key: "last_machine_id", Value: 1000000000},
		})

		servers.Config.LastPID = 500
		servers.Config.LastCharacterID = 0
		servers.Config.LastBandID = 0
		servers.Config.LastSetlistID = 0
		servers.Config.ProfanityList = []string{}
		servers.Config.BattleLimit = 1

		if err != nil {
			log.Fatalln("Could not create default config! GoCentral cannot proceed: ", err)
		}
	}

	// seed randomness with current time
	rand.Seed(time.Now().UnixNano())

	go servers.StartAuthServer()
	go servers.StartSecureServer()

	// Start HTTP server using Chi
	enableRESTAPI := os.Getenv("ENABLERESTAPI")

	httpServer := &http.Server{}

	if enableRESTAPI == "1" {
		r := chi.NewRouter()

		// used to check if the server is up
		r.Get("/health", restapi.HealthHandler)

		// some basic stats about how many chars/bands/scores/etc are in the DB
		// does not include any user-specific information
		r.Get("/stats", restapi.StatsHandler)

		// used to get the current MOTD
		r.Get("/motd", restapi.MotdHandler)

		r.Get("/song_list", restapi.SongListHandler)

		// legacy endpoint, will keep around for now
		r.Get("/leaderboards", restapi.LeaderboardHandler)

		r.Get("/leaderboards/song", restapi.LeaderboardHandler)
		r.Get("/leaderboards/battle", restapi.BattleLeaderboardHandler)

		r.Get("/battles", restapi.BattleListHandler)

		r.Route("/admin", func(r chi.Router) {
			r.Use(restapi.AdminTokenAuth)

			// battle management
			r.Post("/battles/create", restapi.CreateBattleHandler)
			r.Delete("/battles", restapi.DeleteBattleHandler)

			// ban Management
			r.Get("/players/banned", restapi.ListBannedPlayersHandler)
			r.Post("/players/ban", restapi.BanPlayerHandler)
			r.Post("/players/unban", restapi.UnbanPlayerHandler)
			r.Delete("/players/scores", restapi.DeletePlayerScoresHandler)
		})

		httpPort := os.Getenv("HTTPPORT")

		if httpPort == "" {
			log.Printf("REST API enabled but HTTP port, not set, please set an HTTP port using the HTTPPORT environment variable!")
			return
		}

		httpServer = &http.Server{
			Addr:    ":" + httpPort,
			Handler: r,
		}

		go func() {
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Could not listen on :%v: %v\n", httpPort, err)
			}
		}()
		log.Println("GoCentral REST API HTTP server started on:" + httpPort)
	}

	enableHousekeeping := os.Getenv("ENABLEHOUSEKEEPING")

	quit := make(chan struct{})

	if enableHousekeeping == "true" {
		log.Printf("Starting housekeeping tasks...\n")

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		// automatically run some housekeeping tasks
		go func() {
			for {
				select {
				case <-ticker.C:
					database.CleanupDuplicateScores()
					database.PruneOldSessions()
					database.CleanupInvalidScores()
					database.DeleteExpiredBattles()
					database.CleanupBannedUserScores()
					database.CleanupInvalidUsers()
				case <-quit:
					return
				}
			}
		}()
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Printf("Signal (%s) received, stopping\n", s)

	if enableRESTAPI == "true" {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Fatalf("HTTP server Shutdown: %v", err)
		}
	}
	close(quit)
}
