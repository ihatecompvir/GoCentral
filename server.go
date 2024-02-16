package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	database "rb3server/database"
	"rb3server/servers"
)

func main() {
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

	database.GocentralDatabase = client.Database("rockcentral")

	configCollection := database.GocentralDatabase.Collection("config")

	// get config from DB
	err = configCollection.FindOne(nil, bson.M{}).Decode(&servers.Config)
	if err != nil {
		log.Println("Could not get config from MongoDB database, creating default config: ", err)
		_, err = configCollection.InsertOne(nil, bson.D{
			{Key: "last_pid", Value: 500},
			{Key: "last_band_id", Value: 0},
			{Key: "last_character_id", Value: 0},
		})

		servers.Config.LastPID = 500
		servers.Config.LastCharacterID = 0
		servers.Config.LastBandID = 0

		if err != nil {
			log.Fatalln("Could not create default config! GoCentral cannot proceed: ", err)
		}
	}

	// seed randomness with current time
	rand.Seed(time.Now().UnixNano())

	go servers.StartAuthServer()
	go servers.StartSecureServer()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Printf("Signal (%s) received, stopping\n", s)
}
