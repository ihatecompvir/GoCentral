package accomplishment

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"

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

	res, _ := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !res {
		log.Println("Client is attempting to record accomplishments without a valid server-assigned PID, rejecting call")
		return "", nil
	}

	accomplishmentsCollection := database.Collection("accomplishments")

	var accomplishments models.Accomplishments
	err = accomplishmentsCollection.FindOne(nil, bson.M{}).Decode(&accomplishments)

	for idx, entry := range accomplishments.LBGoalValueCampaignMetascore {
		if entry.PID == req.PID {
			if entry.PID == req.PID {
				accomplishments.LBGoalValueCampaignMetascore[idx].Score = req.LBGoalValueCampaignMetascore
				accomplishments.LBGoalValueAccTourgoldlocal1[idx].Score = req.LBGoalValueAccTourgoldlocal1
				accomplishments.LBGoalValueAccTourgoldlocal2[idx].Score = req.LBGoalValueAccTourgoldlocal2
				accomplishments.LBGoalValueAccTourgoldregional1[idx].Score = req.LBGoalValueAccTourgoldregional1
				accomplishments.LBGoalValueAccTourgoldregional2[idx].Score = req.LBGoalValueAccTourgoldregional2
				accomplishments.LBGoalValueAccTourgoldcontinental1[idx].Score = req.LBGoalValueAccTourgoldcontinental1
				accomplishments.LBGoalValueAccTourgoldcontinental2[idx].Score = req.LBGoalValueAccTourgoldcontinental2
				accomplishments.LBGoalValueAccTourgoldcontinental3[idx].Score = req.LBGoalValueAccTourgoldcontinental3
				accomplishments.LBGoalValueAccTourgoldglobal1[idx].Score = req.LBGoalValueAccTourgoldglobal1
				accomplishments.LBGoalValueAccTourgoldglobal2[idx].Score = req.LBGoalValueAccTourgoldglobal2
				accomplishments.LBGoalValueAccTourgoldglobal3[idx].Score = req.LBGoalValueAccTourgoldglobal3
				accomplishments.LBGoalValueAccOverdrivemaintain3[idx].Score = req.LBGoalValueAccOverdrivemaintain3
				accomplishments.LBGoalValueAccOverdrivecareer[idx].Score = req.LBGoalValueAccOverdrivecareer
				accomplishments.LBGoalValueAccCareersaves[idx].Score = req.LBGoalValueAccCareersaves
				accomplishments.LBGoalValueAccMillionpoints[idx].Score = req.LBGoalValueAccMillionpoints
				accomplishments.LBGoalValueAccBassstreaklarge[idx].Score = req.LBGoalValueAccBassstreaklarge
				accomplishments.LBGoalValueAccHopothreehundredbass[idx].Score = req.LBGoalValueAccHopothreehundredbass
				accomplishments.LBGoalValueAccDrumfill170[idx].Score = req.LBGoalValueAccDrumfill170
				accomplishments.LBGoalValueAccDrumstreaklong[idx].Score = req.LBGoalValueAccDrumstreaklong
				accomplishments.LBGoalValueAccDeployguitarfour[idx].Score = req.LBGoalValueAccDeployguitarfour
				accomplishments.LBGoalValueAccGuitarstreaklarge[idx].Score = req.LBGoalValueAccGuitarstreaklarge
				accomplishments.LBGoalValueAccHopoonethousand[idx].Score = req.LBGoalValueAccHopoonethousand
				accomplishments.LBGoalValueAccDoubleawesomealot[idx].Score = req.LBGoalValueAccDoubleawesomealot
				accomplishments.LBGoalValueAccTripleawesomealot[idx].Score = req.LBGoalValueAccTripleawesomealot
				accomplishments.LBGoalValueAccKeystreaklong[idx].Score = req.LBGoalValueAccKeystreaklong
				accomplishments.LBGoalValueAccProbassstreakepic[idx].Score = req.LBGoalValueAccProbassstreakepic
				accomplishments.LBGoalValueAccProdrumroll3[idx].Score = req.LBGoalValueAccProdrumroll3
				accomplishments.LBGoalValueAccProdrumstreaklong[idx].Score = req.LBGoalValueAccProdrumstreaklong
				accomplishments.LBGoalValueAccProguitarstreakepic[idx].Score = req.LBGoalValueAccProguitarstreakepic
				accomplishments.LBGoalValueAccProkeystreaklong[idx].Score = req.LBGoalValueAccProkeystreaklong
				accomplishments.LBGoalValueAccDeployvocals[idx].Score = req.LBGoalValueAccDeployvocals
				accomplishments.LBGoalValueAccDeployvocalsonehundred[idx].Score = req.LBGoalValueAccDeployvocalsonehundred

				goto update
			}
		}
	}

	accomplishments.LBGoalValueCampaignMetascore = append(accomplishments.LBGoalValueCampaignMetascore, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueCampaignMetascore})
	accomplishments.LBGoalValueAccTourgoldlocal1 = append(accomplishments.LBGoalValueAccTourgoldlocal1, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldlocal1})
	accomplishments.LBGoalValueAccTourgoldlocal2 = append(accomplishments.LBGoalValueAccTourgoldlocal2, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldlocal2})
	accomplishments.LBGoalValueAccTourgoldregional1 = append(accomplishments.LBGoalValueAccTourgoldregional1, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldregional1})
	accomplishments.LBGoalValueAccTourgoldregional2 = append(accomplishments.LBGoalValueAccTourgoldregional2, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldregional2})
	accomplishments.LBGoalValueAccTourgoldcontinental1 = append(accomplishments.LBGoalValueAccTourgoldcontinental1, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldcontinental1})
	accomplishments.LBGoalValueAccTourgoldcontinental2 = append(accomplishments.LBGoalValueAccTourgoldcontinental2, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldcontinental2})
	accomplishments.LBGoalValueAccTourgoldcontinental3 = append(accomplishments.LBGoalValueAccTourgoldcontinental3, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldcontinental3})
	accomplishments.LBGoalValueAccTourgoldglobal1 = append(accomplishments.LBGoalValueAccTourgoldglobal1, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldglobal1})
	accomplishments.LBGoalValueAccTourgoldglobal2 = append(accomplishments.LBGoalValueAccTourgoldglobal2, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldglobal2})
	accomplishments.LBGoalValueAccTourgoldglobal3 = append(accomplishments.LBGoalValueAccTourgoldglobal3, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTourgoldglobal3})
	accomplishments.LBGoalValueAccOverdrivemaintain3 = append(accomplishments.LBGoalValueAccOverdrivemaintain3, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccOverdrivemaintain3})
	accomplishments.LBGoalValueAccOverdrivecareer = append(accomplishments.LBGoalValueAccOverdrivecareer, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccOverdrivecareer})
	accomplishments.LBGoalValueAccCareersaves = append(accomplishments.LBGoalValueAccCareersaves, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccCareersaves})
	accomplishments.LBGoalValueAccMillionpoints = append(accomplishments.LBGoalValueAccMillionpoints, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccMillionpoints})
	accomplishments.LBGoalValueAccBassstreaklarge = append(accomplishments.LBGoalValueAccBassstreaklarge, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccBassstreaklarge})
	accomplishments.LBGoalValueAccHopothreehundredbass = append(accomplishments.LBGoalValueAccHopothreehundredbass, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccHopothreehundredbass})
	accomplishments.LBGoalValueAccDrumfill170 = append(accomplishments.LBGoalValueAccDrumfill170, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccDrumfill170})
	accomplishments.LBGoalValueAccDrumstreaklong = append(accomplishments.LBGoalValueAccDrumstreaklong, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccDrumstreaklong})
	accomplishments.LBGoalValueAccDeployguitarfour = append(accomplishments.LBGoalValueAccDeployguitarfour, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccDeployguitarfour})
	accomplishments.LBGoalValueAccGuitarstreaklarge = append(accomplishments.LBGoalValueAccGuitarstreaklarge, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccGuitarstreaklarge})
	accomplishments.LBGoalValueAccHopoonethousand = append(accomplishments.LBGoalValueAccHopoonethousand, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccHopoonethousand})
	accomplishments.LBGoalValueAccDoubleawesomealot = append(accomplishments.LBGoalValueAccDoubleawesomealot, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccDoubleawesomealot})
	accomplishments.LBGoalValueAccTripleawesomealot = append(accomplishments.LBGoalValueAccTripleawesomealot, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccTripleawesomealot})
	accomplishments.LBGoalValueAccKeystreaklong = append(accomplishments.LBGoalValueAccKeystreaklong, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccKeystreaklong})
	accomplishments.LBGoalValueAccProbassstreakepic = append(accomplishments.LBGoalValueAccProbassstreakepic, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccProbassstreakepic})
	accomplishments.LBGoalValueAccProdrumroll3 = append(accomplishments.LBGoalValueAccProdrumroll3, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccProdrumroll3})
	accomplishments.LBGoalValueAccProdrumstreaklong = append(accomplishments.LBGoalValueAccProdrumstreaklong, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccProdrumstreaklong})
	accomplishments.LBGoalValueAccProguitarstreakepic = append(accomplishments.LBGoalValueAccProguitarstreakepic, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccProguitarstreakepic})
	accomplishments.LBGoalValueAccProkeystreaklong = append(accomplishments.LBGoalValueAccProkeystreaklong, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccProkeystreaklong})
	accomplishments.LBGoalValueAccDeployvocals = append(accomplishments.LBGoalValueAccDeployvocals, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccDeployvocals})
	accomplishments.LBGoalValueAccDeployvocalsonehundred = append(accomplishments.LBGoalValueAccDeployvocalsonehundred, models.AccomplishmentScoreEntry{req.PID, req.LBGoalValueAccDeployvocalsonehundred})

update:
	update := bson.M{
		"$set": bson.M{
			"lb_goal_value_campaign_metascore":         accomplishments.LBGoalValueCampaignMetascore,
			"lb_goal_value_acc_tourgoldlocal1":         accomplishments.LBGoalValueAccTourgoldlocal1,
			"lb_goal_value_acc_tourgoldlocal2":         accomplishments.LBGoalValueAccTourgoldlocal2,
			"lb_goal_value_acc_tourgoldregional1":      accomplishments.LBGoalValueAccTourgoldregional1,
			"lb_goal_value_acc_tourgoldregional2":      accomplishments.LBGoalValueAccTourgoldregional2,
			"lb_goal_value_acc_tourgoldcontinental1":   accomplishments.LBGoalValueAccTourgoldcontinental1,
			"lb_goal_value_acc_tourgoldcontinental2":   accomplishments.LBGoalValueAccTourgoldcontinental2,
			"lb_goal_value_acc_tourgoldcontinental3":   accomplishments.LBGoalValueAccTourgoldcontinental3,
			"lb_goal_value_acc_tourgoldglobal1":        accomplishments.LBGoalValueAccTourgoldglobal1,
			"lb_goal_value_acc_tourgoldglobal2":        accomplishments.LBGoalValueAccTourgoldglobal2,
			"lb_goal_value_acc_tourgoldglobal3":        accomplishments.LBGoalValueAccTourgoldglobal3,
			"lb_goal_value_acc_overdrivemaintain3":     accomplishments.LBGoalValueAccOverdrivemaintain3,
			"lb_goal_value_acc_overdrivecareer":        accomplishments.LBGoalValueAccOverdrivecareer,
			"lb_goal_value_acc_careersaves":            accomplishments.LBGoalValueAccCareersaves,
			"lb_goal_value_acc_millionpoints":          accomplishments.LBGoalValueAccMillionpoints,
			"lb_goal_value_acc_bassstreaklarge":        accomplishments.LBGoalValueAccBassstreaklarge,
			"lb_goal_value_acc_hopothreehundredbass":   accomplishments.LBGoalValueAccHopothreehundredbass,
			"lb_goal_value_acc_drumfill170":            accomplishments.LBGoalValueAccDrumfill170,
			"lb_goal_value_acc_drumstreaklong":         accomplishments.LBGoalValueAccDrumstreaklong,
			"lb_goal_value_acc_deployguitarfour":       accomplishments.LBGoalValueAccDeployguitarfour,
			"lb_goal_value_acc_guitarstreaklarge":      accomplishments.LBGoalValueAccGuitarstreaklarge,
			"lb_goal_value_acc_hopoonethousand":        accomplishments.LBGoalValueAccHopoonethousand,
			"lb_goal_value_acc_doubleawesomealot":      accomplishments.LBGoalValueAccDoubleawesomealot,
			"lb_goal_value_acc_tripleawesomealot":      accomplishments.LBGoalValueAccTripleawesomealot,
			"lb_goal_value_acc_keystreaklong":          accomplishments.LBGoalValueAccKeystreaklong,
			"lb_goal_value_acc_probassstreakepic":      accomplishments.LBGoalValueAccProbassstreakepic,
			"lb_goal_value_acc_prodrumroll3":           accomplishments.LBGoalValueAccProdrumroll3,
			"lb_goal_value_acc_prodrumstreaklong":      accomplishments.LBGoalValueAccProdrumstreaklong,
			"lb_goal_value_acc_proguitarstreakepic":    accomplishments.LBGoalValueAccProguitarstreakepic,
			"lb_goal_value_acc_prokeystreaklong":       accomplishments.LBGoalValueAccProkeystreaklong,
			"lb_goal_value_acc_deployvocals":           accomplishments.LBGoalValueAccDeployvocals,
			"lb_goal_value_acc_deployvocalsonehundred": accomplishments.LBGoalValueAccDeployvocalsonehundred,
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err = accomplishmentsCollection.UpdateOne(context.TODO(), bson.M{}, update, opts)

	if err != nil {
		log.Printf("Could not update accomplishments for PID %v: %s\n", req.PID, err)
		return "", err
	}

	return marshaler.MarshalResponse(service.Path(), []AccomplishmentRecordResponse{{1}})
}
