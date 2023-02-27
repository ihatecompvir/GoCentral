package accomplishment

import (
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AccomplishmentRecordRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`

	// there is more to this request but for now only extract the leaderboard values
	// TODO: add the non-LB stuff here

	LBGoalValueCampaignMetascore         int `json:"lb_goal_value_campaign_metascore"`
	LBGoalValueAccTourgoldlocal1         int `json:"lb_goal_value_acc_tourgoldlocal1"`
	LBGoalValueAccTourgoldlocal2         int `json:"lb_goal_value_acc_tourgoldlocal2"`
	LBGoalValueAccTourgoldregional1      int `json:"lb_goal_value_acc_tourgoldregional1"`
	LBGoalValueAccTourgoldregional2      int `json:"lb_goal_value_acc_tourgoldregional2"`
	LBGoalValueAccTourgoldcontinental1   int `json:"lb_goal_value_acc_tourgoldcontinental1"`
	LBGoalValueAccTourgoldcontinental2   int `json:"lb_goal_value_acc_tourgoldcontinental2"`
	LBGoalValueAccTourgoldcontinental3   int `json:"lb_goal_value_acc_tourgoldcontinental3"`
	LBGoalValueAccTourgoldglobal1        int `json:"lb_goal_value_acc_tourgoldglobal1"`
	LBGoalValueAccTourgoldglobal2        int `json:"lb_goal_value_acc_tourgoldglobal2"`
	LBGoalValueAccTourgoldglobal3        int `json:"lb_goal_value_acc_tourgoldglobal3"`
	LBGoalValueAccOverdrivemaintain3     int `json:"lb_goal_value_acc_overdrivemaintain3"`
	LBGoalValueAccOverdrivecareer        int `json:"lb_goal_value_acc_overdrivecareer"`
	LBGoalValueAccCareersaves            int `json:"lb_goal_value_acc_careersaves"`
	LBGoalValueAccMillionpoints          int `json:"lb_goal_value_acc_millionpoints"`
	LBGoalValueAccBassstreaklarge        int `json:"lb_goal_value_acc_bassstreaklarge"`
	LBGoalValueAccHopothreehundredbass   int `json:"lb_goal_value_acc_hopothreehundredbass"`
	LBGoalValueAccDrumfill170            int `json:"lb_goal_value_acc_drumfill170"`
	LBGoalValueAccDrumstreaklong         int `json:"lb_goal_value_acc_drumstreaklong"`
	LBGoalValueAccDeployguitarfour       int `json:"lb_goal_value_acc_deployguitarfour"`
	LBGoalValueAccGuitarstreaklarge      int `json:"lb_goal_value_acc_guitarstreaklarge"`
	LBGoalValueAccHopoonethousand        int `json:"lb_goal_value_acc_hopoonethousand"`
	LBGoalValueAccDoubleawesomealot      int `json:"lb_goal_value_acc_doubleawesomealot"`
	LBGoalValueAccTripleawesomealot      int `json:"lb_goal_value_acc_tripleawesomealot"`
	LBGoalValueAccKeystreaklong          int `json:"lb_goal_value_acc_keystreaklong"`
	LBGoalValueAccProbassstreakepic      int `json:"lb_goal_value_acc_probassstreakepic"`
	LBGoalValueAccProdrumroll3           int `json:"lb_goal_value_acc_prodrumroll3"`
	LBGoalValueAccProdrumstreaklong      int `json:"lb_goal_value_acc_prodrumstreaklong"`
	LBGoalValueAccProguitarstreakepic    int `json:"lb_goal_value_acc_proguitarstreakepic"`
	LBGoalValueAccProkeystreaklong       int `json:"lb_goal_value_acc_prokeystreaklong"`
	LBGoalValueAccDeployvocals           int `json:"lb_goal_value_acc_deployvocals"`
	LBGoalValueAccDeployvocalsonehundred int `json:"lb_goal_value_acc_deployvocalsonehundred"`
}

type AccomplishmentRecordResponse struct {
	Success int `json:"success"`
}

type AccomplishmentRecordService struct {
}

func (service AccomplishmentRecordService) Path() string {
	return "accomplishment/record"
}

func (service AccomplishmentRecordService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req AccomplishmentRecordRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for recording accomplishment")
		return "", err
	}

	accomplishmentsCollection := database.Collection("accomplishments")

	var accomplishments models.Accomplishments
	err = accomplishmentsCollection.FindOne(nil, bson.M{"pid": req.PID}).Decode(&accomplishments)

	filter := bson.D{{"pid", req.PID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "lb_goal_value_campaign_metascore", Value: req.LBGoalValueCampaignMetascore},
			{Key: "lb_goal_value_acc_tourgoldlocal1", Value: req.LBGoalValueAccTourgoldlocal1},
			{Key: "lb_goal_value_acc_tourgoldlocal2", Value: req.LBGoalValueAccTourgoldlocal2},
			{Key: "lb_goal_value_acc_tourgoldregional1", Value: req.LBGoalValueAccTourgoldregional1},
			{Key: "lb_goal_value_acc_tourgoldregional2", Value: req.LBGoalValueAccTourgoldregional2},
			{Key: "lb_goal_value_acc_tourgoldcontinental1", Value: req.LBGoalValueAccTourgoldcontinental1},
			{Key: "lb_goal_value_acc_tourgoldcontinental2", Value: req.LBGoalValueAccTourgoldcontinental2},
			{Key: "lb_goal_value_acc_tourgoldcontinental3", Value: req.LBGoalValueAccTourgoldcontinental3},
			{Key: "lb_goal_value_acc_tourgoldglobal1", Value: req.LBGoalValueAccTourgoldglobal1},
			{Key: "lb_goal_value_acc_tourgoldglobal2", Value: req.LBGoalValueAccTourgoldglobal2},
			{Key: "lb_goal_value_acc_tourgoldglobal3", Value: req.LBGoalValueAccTourgoldglobal3},
			{Key: "lb_goal_value_acc_overdrivemaintain3", Value: req.LBGoalValueAccOverdrivemaintain3},
			{Key: "lb_goal_value_acc_overdrivecareer", Value: req.LBGoalValueAccOverdrivecareer},
			{Key: "lb_goal_value_acc_careersaves", Value: req.LBGoalValueAccCareersaves},
			{Key: "lb_goal_value_acc_millionpoints", Value: req.LBGoalValueAccMillionpoints},
			{Key: "lb_goal_value_acc_bassstreaklarge", Value: req.LBGoalValueAccBassstreaklarge},
			{Key: "lb_goal_value_acc_hopothreehundredbass", Value: req.LBGoalValueAccHopothreehundredbass},
			{Key: "lb_goal_value_acc_drumfill170", Value: req.LBGoalValueAccDrumfill170},
			{Key: "lb_goal_value_acc_drumstreaklong", Value: req.LBGoalValueAccDrumstreaklong},
			{Key: "lb_goal_value_acc_deployguitarfour", Value: req.LBGoalValueAccDeployguitarfour},
			{Key: "lb_goal_value_acc_guitarstreaklarge", Value: req.LBGoalValueAccGuitarstreaklarge},
			{Key: "lb_goal_value_acc_hopoonethousand", Value: req.LBGoalValueAccHopoonethousand},
			{Key: "lb_goal_value_acc_doubleawesomealot", Value: req.LBGoalValueAccDoubleawesomealot},
			{Key: "lb_goal_value_acc_tripleawesomealot", Value: req.LBGoalValueAccTripleawesomealot},
			{Key: "lb_goal_value_acc_keystreaklong", Value: req.LBGoalValueAccKeystreaklong},
			{Key: "lb_goal_value_acc_probassstreakepic", Value: req.LBGoalValueAccProbassstreakepic},
			{Key: "lb_goal_value_acc_prodrumroll3", Value: req.LBGoalValueAccProdrumroll3},
			{Key: "lb_goal_value_acc_prodrumstreaklong", Value: req.LBGoalValueAccProdrumstreaklong},
			{Key: "lb_goal_value_acc_proguitarstreakepic", Value: req.LBGoalValueAccProguitarstreakepic},
			{Key: "lb_goal_value_acc_prokeystreaklong", Value: req.LBGoalValueAccProkeystreaklong},
			{Key: "lb_goal_value_acc_deployvocals", Value: req.LBGoalValueAccDeployvocals},
			{Key: "lb_goal_value_acc_deployvocalsonehundred", Value: req.LBGoalValueAccDeployvocalsonehundred},
		}},
	}
	opts := options.Update().SetUpsert(true)
	_, err = accomplishmentsCollection.UpdateOne(nil, filter, update, opts)

	if err != nil {
		log.Printf("Could not update accomplishments for PID %v: %s\n", req.PID, err)
		return "", err
	}

	return marshaler.MarshalResponse(service.Path(), []AccomplishmentRecordResponse{{1}})
}
