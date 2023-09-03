package leaderboard

import (
	"context"
	"fmt"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/knvtva/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PlayerGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	SongID      int    `json:"song_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
	RoleID      int    `json:"role_id"`
	LBType      int    `json:"lb_type"`
	LBMode      int    `json:"lb_mode"`
	NumRows     int    `json:"num_rows"`
}

type PlayerGetResponse struct {
	PID          int    `json:"pid"`
	Name         string `json:"name"`
	DiffID       int    `json:"diff_id"`
	Rank         int    `json:"rank"`
	Score        int    `json:"score"`
	IsPercentile int    `json:"is_percentile"`
	InstMask     int    `json:"inst_mask"`
	NotesPct     int    `json:"notes_pct"`
	IsFriend     int    `json:"is_friend"`
	UnnamedBand  int    `json:"unnamed_band"`
	PGUID        string `json:"pguid"`
	ORank        int    `json:"orank"`
}

var instrumentMap = map[int]int{
	0: 1,
	1: 2,
	2: 4,
	3: 8,
	4: 16,
	5: 32,
	6: 64,
	7: 128,
	8: 256,
	9: 512,
}

type PlayerGetService struct {
}

var debugging bool = true

func (service PlayerGetService) Path() string {
	return "leaderboards/player/get"
}

func (service PlayerGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req PlayerGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		users := database.Collection("users")
		var user models.User
		err = users.FindOne(nil, bson.M{"pid": req.PID000}).Decode(&user)
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for leaderboards")
		log.Println("Database PID : ", user.PID)
		client.SetPlayerID(user.PID)
		log.Println("Client PID : ", client.PlayerID())
	}

	scoresCollection := database.Collection("scores")

	var playerPosition int64 // where the player is on the leaderboards
	var scoresToSkip int64   // how many scores to skip to get to the player's rank
	var startIndex int
	var playerHasScore bool = false
	var curIndex int

	log.Println("Looking for scores")

	// First, get the player's score
	// This will be used to find where the player is at on the leaderboards
	playerFilter := bson.M{"song_id": req.SongID, "role_id": req.RoleID, "pid": req.PID000}
	var playerScore models.Score
	err = scoresCollection.FindOne(context.TODO(), playerFilter).Decode(&playerScore)
	if err != nil {
		// the player isn't on the leaderboards, so we just start from #1
		playerPosition = 1
		scoresToSkip = 0
		startIndex = 1
		playerHasScore = false
	} else {
		// find the player's position on the leaderboards
		playerPosition, err = scoresCollection.CountDocuments(context.TODO(), bson.M{"song_id": req.SongID, "role_id": req.RoleID, "score": bson.M{"$gt": playerScore.Score}})
		playerHasScore = true
		if err != nil {
			// something went wrong so just get #1
			playerPosition = 1
			scoresToSkip = 0
			startIndex = 1
			playerHasScore = false
		}
	}
	// get the name of the currently logged in player
	var loggedinuser string
	users := database.Collection("users")
	var theusers models.User
	err = users.FindOne(nil, bson.M{"pid": req.PID000}).Decode(&theusers)
	if err == nil {
		loggedinuser = theusers.Username
	}

	log.Println(loggedinuser)

	log.Println("Player position is : ", playerPosition)

	// get all scores for the song and role ID
	// skipping ahead by the player's position on the leaderboards
	// sorting by score descending
	// limiting to the number of scores requested
	filter := bson.M{"song_id": req.SongID, "role_id": req.RoleID}
	cur, err := scoresCollection.Find(context.TODO(), filter, options.Find().
		SetLimit(int64(req.NumRows)).
		SetSkip(scoresToSkip).
		SetSort(bson.D{{"score", -1}}))

	if err != nil {
		// we couldn't get any scores, so just fallback to a blank response
		return "", err
	}

	res := []PlayerGetResponse{}

	// used to calculate rank
	if playerHasScore {
		curIndex = startIndex + 1
	} else {
		curIndex = 1
	}

	// use the cursor to read every score and append it to the response
	for cur.Next(nil) {
		username := "Player"

		// decode the score into a score object
		var score models.Score
		var createUserName string
		err := cur.Decode(&score)
		if err != nil {
			// we couldn't decode the score, so just fallback to a blank response
			log.Printf("Error decoding score: %v", err)
			return marshaler.MarshalResponse(service.Path(), []PlayerGetResponse{{}})
		}

		// BOI = "band or instrument" presumably, so detect if we're looking up a band score or an instrument score
		// role ID 10 == band role
		if score.BOI == 1 && req.RoleID != 10 {

			users := database.Collection("users")
			var user models.User
			err = users.FindOne(nil, bson.M{"pid": score.OwnerPID}).Decode(&user)

			if err == nil {
				username = user.Username
			}
			createUserName = username
			if debugging {

				log.Println("Owner pid : ", score.OwnerPID)
				log.Println("Username : ", username)
				log.Println("Difficulty : ", score.DiffID)
				log.Println("Current index : ", curIndex)
				switch user.ConsoleType {
				case 0:
					log.Println("Machine type - Xbox 360")
				case 1:
					log.Println("Machine type - PS3")
				case 2:
					log.Println("Machine type - Wii")
				default:
					log.Println("Machine type - unknown")
				}
				log.Println("Score : ", score.Score)
				log.Println("Instrument mask : ", score.InstrumentMask)
				log.Println("Note percentage : ", score.NotesPercent)
			}
			if user.ConsoleType == 0 {
				createUserName = createUserName + " - XBOX360"
			} else if user.ConsoleType == 1 {
				createUserName = createUserName + " - PS3"
			} else if user.ConsoleType == 2 {
				createUserName = createUserName + " - Wii"
			}
			if score.OwnerPID > 500 && score.Score != 0 {
				log.Println("Adding this score to res as owner > 500 ")
				res = append(res, PlayerGetResponse{
					score.OwnerPID,
					createUserName,
					score.DiffID,
					curIndex,
					score.Score,
					0,
					instrumentMap[score.RoleID],
					score.NotesPercent,
					1,
					0,
					"N/A", // this is what the official servers used
					curIndex,
				})
			}

		} else {
			// its a band score, so get the band name so it can appear properly on the leaderboard

			users := database.Collection("users")
			var bandUser models.User
			err = users.FindOne(nil, bson.M{"pid": score.OwnerPID}).Decode(&bandUser)

			username = bandUser.Username

			bands := database.Collection("bands")
			var band models.Band
			bandName := fmt.Sprintf("%v's Band", username)
			err = bands.FindOne(nil, bson.M{"owner_pid": score.OwnerPID}).Decode(&band)

			if err == nil {
				bandName = band.Name
			}

			createUserName = username

			if bandUser.ConsoleType == 0 {
				createUserName = createUserName + " - XBOX360"
			} else if bandUser.ConsoleType == 1 {
				createUserName = createUserName + " - PS3"
			} else if bandUser.ConsoleType == 2 {
				createUserName = createUserName + " - Wii"
			}

			if score.RoleID != 10 {
				res = append(res, PlayerGetResponse{
					score.OwnerPID,
					createUserName,
					score.DiffID,
					curIndex,
					score.Score,
					0,
					instrumentMap[score.RoleID],
					score.NotesPercent,
					1,
					0,
					"N/A",
					curIndex,
				})
			} else {
				res = append(res, PlayerGetResponse{
					score.OwnerPID,
					bandName,
					score.DiffID,
					curIndex,
					score.Score,
					0,
					score.InstrumentMask,
					score.NotesPercent,
					1,
					0,
					"N/A",
					curIndex,
			})
		}
			if debugging {

				log.Println("Owner pid : ", score.OwnerPID)
				log.Println("Band name : ", bandName)
				log.Println("Difficulty : ", score.DiffID)
				log.Println("Current index : ", curIndex)
				switch band.ConsoleType {
				case 0:
					log.Println("Machine type - Xbox 360")
				case 1:
					log.Println("Machine type - PS3")
				case 2:
					log.Println("Machine type - Wii")
				default:
					log.Println("Machine type - unknown")
				}
				log.Println("Score : ", score.Score)
				log.Println("Instrument mask : ", score.InstrumentMask)
				log.Println("Note percentage : ", score.NotesPercent)
			}
			log.Print(score.OwnerPID)
			log.Print(bandName)
			log.Print(score.DiffID)
			log.Print(curIndex)
			log.Print(score.Score)
			log.Print(score.InstrumentMask)
			log.Print(score.NotesPercent)
			log.Print(curIndex)

		}
		curIndex += 1
	}

	if len(res) == 0 {
		return "", err
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
