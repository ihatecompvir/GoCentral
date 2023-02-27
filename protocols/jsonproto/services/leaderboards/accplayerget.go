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

type AccPlayerGetRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	AccID       string `json:"acc_id"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID000      int    `json:"pid000"`
}

type AccPlayerGetResponse struct {
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

type AccPlayerGetService struct {
}

// a function that gets the acc_id and returns the proper field of the Accomplishments model
// for example, campaign_metascore is LBGoalValueCampaignMetascore in models.Accomplishments
func getAccomplishmentField(accID string, accomplishments models.Accomplishments) int {
	switch accID {
	case "campaign_metascore":
		return accomplishments.LBGoalValueCampaignMetascore
	case "acc_tourgoldlocal1":
		return accomplishments.LBGoalValueAccTourgoldlocal1
	case "acc_tourgoldlocal2":
		return accomplishments.LBGoalValueAccTourgoldlocal2
	case "acc_tourgoldregional1":
		return accomplishments.LBGoalValueAccTourgoldregional1
	case "acc_tourgoldregional2":
		return accomplishments.LBGoalValueAccTourgoldregional2
	case "acc_tourgoldcontinental1":
		return accomplishments.LBGoalValueAccTourgoldcontinental1
	case "acc_tourgoldcontinental2":
		return accomplishments.LBGoalValueAccTourgoldcontinental2
	case "acc_tourgoldglobal1":
		return accomplishments.LBGoalValueAccTourgoldglobal1
	case "acc_tourgoldglobal2":
		return accomplishments.LBGoalValueAccTourgoldglobal2
	case "acc_tourgoldglobal3":
		return accomplishments.LBGoalValueAccTourgoldglobal3
	case "acc_overdrivemaintain3":
		return accomplishments.LBGoalValueAccOverdrivemaintain3
	case "acc_overdrivecareer":
		return accomplishments.LBGoalValueAccOverdrivecareer
	case "acc_careersaves":
		return accomplishments.LBGoalValueAccCareersaves
	case "acc_millionpoints":
		return accomplishments.LBGoalValueAccMillionpoints
	case "acc_bassstreaklarge":
		return accomplishments.LBGoalValueAccBassstreaklarge
	case "acc_hopothreehundredbass":
		return accomplishments.LBGoalValueAccHopothreehundredbass
	case "acc_drumfill170":
		return accomplishments.LBGoalValueAccDrumfill170
	case "acc_drumstreaklong":
		return accomplishments.LBGoalValueAccDrumstreaklong
	case "acc_deployguitarfour":
		return accomplishments.LBGoalValueAccDeployguitarfour
	case "acc_guitarstreaklarge":
		return accomplishments.LBGoalValueAccGuitarstreaklarge
	case "acc_keystreaklong":
		return accomplishments.LBGoalValueAccKeystreaklong
	case "acc_hopoonethousand":
		return accomplishments.LBGoalValueAccHopoonethousand
	case "acc_doubleawesomealot":
		return accomplishments.LBGoalValueAccDoubleawesomealot
	case "acc_tripleawesomealot":
		return accomplishments.LBGoalValueAccTripleawesomealot
	case "acc_probassstreakepic":
		return accomplishments.LBGoalValueAccProbassstreakepic
	case "acc_prodrumroll3":
		return accomplishments.LBGoalValueAccProdrumroll3
	case "acc_prodrumstreaklong":
		return accomplishments.LBGoalValueAccProdrumstreaklong
	case "acc_proguitarstreakepic":
		return accomplishments.LBGoalValueAccProguitarstreakepic
	case "acc_prokeystreaklong":
		return accomplishments.LBGoalValueAccProkeystreaklong
	case "acc_deployvocals":
		return accomplishments.LBGoalValueAccDeployvocals
	case "acc_deployvocalsonehundred":
		return accomplishments.LBGoalValueAccDeployvocalsonehundred
	default:
		return 0
	}
}

func (service AccPlayerGetService) Path() string {
	return "leaderboards/acc_player/get"
}

func (service AccPlayerGetService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccPlayerGetRequest

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

	cur, err := accomplishmentsCollection.Find(context.TODO(), bson.D{}, options.Find().SetLimit(20).SetSort(bson.D{{"lb_goal_value_" + req.AccID, -1}}))
	if err != nil {
		log.Printf("Could not find accomplishments for %v: %v", req.AccID, err)
		return "", err
	}

	res := []AccPlayerGetResponse{}

	curIndex := 1

	for cur.Next(nil) {
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
			log.Printf("Could not find user with PID %d, defaulting to \"Player\": %v", accomplishments.PID, err)
		}

		if user.Username != "" {
			username = user.Username
		} else {
			username = "Player"
		}

		res = append(res, AccPlayerGetResponse{
			accomplishments.PID,
			username,
			0,
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
