package leaderboard

import (
	"context"
	"log"
	db "rb3server/database"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"sort"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
func getAccomplishmentField(accID string, accomplishments models.Accomplishments) []models.AccomplishmentScoreEntry {
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
		return []models.AccomplishmentScoreEntry{}
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
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for acc leaderboards")
		return "", err
	}

	accomplishmentsCollection := database.Collection("accomplishments")

	// FindOne the accomplishment scores
	var accomplishments models.Accomplishments
	err = accomplishmentsCollection.FindOne(context.TODO(), bson.M{}).Decode(&accomplishments)

	if err != nil {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	}

	res := []AccPlayerGetResponse{}

	accSlice := getAccomplishmentField(req.AccID, accomplishments)

	// sort acc scores by score
	sort.Slice(accSlice, func(i, j int) bool {
		return accSlice[i].Score > accSlice[j].Score
	})

	// find the player's score idx in the sorted list
	// if the player has no scores, just start with the first score
	playerScoreIdx := 0
	for idx, score := range accSlice {
		if score.PID == req.PID000 {
			playerScoreIdx = idx
			break
		}
	}

	// start and end idx must be in a window size of 20 otherwise the UI will act a bit buggy
	startIdx := (playerScoreIdx / 20) * 20
	endIdx := min(len(accSlice), startIdx+20)

	for i := startIdx; i < endIdx; i++ {
		score := accSlice[i]
		res = append(res, AccPlayerGetResponse{
			PID:          score.PID,
			Score:        score.Score,
			DiffID:       0,
			Name:         db.GetConsolePrefixedUsernameForPID(score.PID),
			IsPercentile: 0,
			IsFriend:     0,
			InstMask:     0,
			NotesPct:     0,
			UnnamedBand:  0,
			PGUID:        "",
			Rank:         i + 1,
			ORank:        i + 1,
		})
	}

	if len(res) == 0 {
		return marshaler.GenerateEmptyJSONResponse(service.Path()), nil
	} else {
		return marshaler.MarshalResponse(service.Path(), res)
	}
}
