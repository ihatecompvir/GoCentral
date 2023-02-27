package leaderboard

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AccMaxrankGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	AccID       string `json:"acc_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
}

type AccMaxrankGetResponse struct {
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

type AccMaxrankGetService struct {
}

func (service AccMaxrankGetService) Path() string {
	return "leaderboards/acc_maxrank/get"
}

func (service AccMaxrankGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccMaxrankGetRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID000 != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for accomplishment leaderboards")
		return "", err
	}

	accomplishmentsCollection := database.Collection("accomplishments")
	usersCollection := database.Collection("users")

	cur, err := accomplishmentsCollection.Find(context.TODO(), bson.D{}, options.Find().SetLimit(10).SetSort(bson.D{{"lb_goal_value_" + req.AccID, -1}}))
	if err != nil {
		log.Printf("Could not find accomplishments for %v: %v", req.AccID, err)
		return "", err
	}

	res := []AccMaxrankGetResponse{}

	curIndex := 1

	for cur.Next(nil) && curIndex != 16 {
		username := "Player"

		// create a value into which the single document can be decoded
		var accomplishments models.Accomplishments
		err := cur.Decode(&accomplishments)
		if err != nil {
			log.Printf("Could not decode accomplishments: %v", err)
			return "", err
		}

		var user models.User
		err = usersCollection.FindOne(nil, bson.M{"pid": accomplishments.PID}).Decode(&user)

		if err != nil {
			log.Printf("Could not find user with PID %d: %v", accomplishments.PID, err)
		}

		if user.Username != "" {
			username = user.Username
		} else {
			log.Printf("Could not find user with PID %d, defaulting to \"Player\": %v", accomplishments.PID, err)
			username = "Player"
		}

		res = append(res, AccMaxrankGetResponse{
			accomplishments.PID,
			username,
			4,
			curIndex,
			getAccomplishmentField(req.AccID, accomplishments),
			0,
			0,
			100,
			0,
			0,
			"",
			1,
		})

		curIndex++
	}

	return marshaler.MarshalResponse(service.Path(), res)
}
