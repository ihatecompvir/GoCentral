package scores

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"sort"
	"strconv"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BattleScoreRecordRequest struct {
	Region      string `json:"region"`
	Score       int    `json:"score"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PIDs        []int  `json:"pidXXX"`
	BattleID    int    `json:"battle_id"`
	Slots       []int  `json:"slotXXX"`
}

type BattleScoreRecordResponse struct {
	ID           int    `json:"id"`
	IsBOI        int    `json:"is_boi"`
	InstaRank    int    `json:"insta_rank"`
	IsPercentile int    `json:"is_percentile"`
	Part1        string `json:"part_1"`
	Part2        string `json:"part_2"`
}

type BattleScoreRecordService struct {
}

func (service BattleScoreRecordService) Path() string {
	return "battles/record"
}

func (service BattleScoreRecordService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req BattleScoreRecordRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PIDs[0] != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting setlist update")
		return "", err
	}

	setlistCollection := database.Collection("setlists")

	// Find the setlist with the equivalent battle ID to make sure it exists
	var setlist models.Setlist
	err = setlistCollection.FindOne(context.TODO(), bson.M{"setlist_id": req.BattleID}).Decode(&setlist)
	if err != nil {
		log.Printf("Could not find setlist with battle ID %d: %v", req.BattleID, err)
		return "[]", nil
	}

	var score models.BattleScoreEntry
	score.Score = req.Score
	score.PID = req.PIDs[0]

	scoreHigher := []bool{}
	currentScore := []int{}

	// Retrieve the existing score for the PID
	for _, pid := range req.PIDs {
		found := false
		var existingScore models.BattleScoreEntry
		for _, battleScore := range setlist.BattleScores {
			if battleScore.PID == pid {
				existingScore = battleScore
				found = true
				break
			}
		}

		isNewScoreHigher := !found || score.Score > existingScore.Score
		scoreHigher = append(scoreHigher, isNewScoreHigher)

		if isNewScoreHigher {
			filter := bson.M{
				"setlist_id":        req.BattleID,
				"battle_scores.pid": pid,
			}
			update := bson.M{
				"$set": bson.M{"battle_scores.$.score": score.Score},
			}
			result, err := setlistCollection.UpdateOne(context.TODO(), filter, update)

			if err != nil {
				log.Printf("Error updating score for PID %d in battle ID %d: %v", pid, req.BattleID, err)
				return "[]", nil
			}

			// If no score was updated, push the new score
			if result.MatchedCount == 0 {
				filter = bson.M{"setlist_id": req.BattleID}
				update = bson.M{
					"$push": bson.M{"battle_scores": score},
				}
				_, err = setlistCollection.UpdateOne(context.TODO(), filter, update, options.Update().SetUpsert(true))
				if err != nil {
					log.Printf("Could not update setlist with battle ID %d: %v", req.BattleID, err)
					return "[]", nil
				}
			}

			currentScore = append(currentScore, score.Score)
		} else {
			currentScore = append(currentScore, existingScore.Score)
		}
	}

	res := []BattleScoreRecordResponse{}

	err = setlistCollection.FindOne(context.TODO(), bson.M{"setlist_id": req.BattleID}).Decode(&setlist)

	// Sort battle scores by score
	sort.Slice(setlist.BattleScores, func(i, j int) bool {
		return setlist.BattleScores[i].Score > setlist.BattleScores[j].Score
	})

	numPids := len(req.PIDs)

	for i := 0; i < (numPids / 2); i++ {
		playerScoreIdx := 0
		for idx, score := range setlist.BattleScores {
			if score.PID == req.PIDs[i] {
				playerScoreIdx = idx
				break
			}
		}

		if scoreHigher[i] {
			instarank := BattleScoreRecordResponse{
				req.BattleID,
				0,
				int(playerScoreIdx + 1),
				0,
				"b",
				"f",
			}
			res = append(res, instarank)
		} else {
			instarank := BattleScoreRecordResponse{
				req.BattleID,
				0,
				int(playerScoreIdx + 1),
				0,
				"c|" + strconv.Itoa(currentScore[i]),
				"f",
			}
			res = append(res, instarank)
		}
	}

	for i := numPids / 2; i < numPids; i++ {
		playerScoreIdx := 0
		for idx, score := range setlist.BattleScores {
			if score.PID == req.PIDs[i] {
				playerScoreIdx = idx
				break
			}
		}

		if scoreHigher[i] {
			instarank := BattleScoreRecordResponse{
				req.BattleID,
				1,
				int(playerScoreIdx + 1),
				0,
				"b",
				"f",
			}
			res = append(res, instarank)
		} else {
			instarank := BattleScoreRecordResponse{
				req.BattleID,
				1,
				int(playerScoreIdx + 1),
				0,
				"c|" + strconv.Itoa(currentScore[i]),
				"f",
			}
			res = append(res, instarank)
		}
	}

	return marshaler.MarshalResponse(service.Path(), res)
}
