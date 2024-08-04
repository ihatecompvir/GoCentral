package scores

import (
	"context"
	"log"
	db "rb3server/database"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
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

	// make sure the pids array is not empty
	if len(req.PIDs) == 0 {
		log.Println("PID array is empty, rejecting battle score record")
		return "", err
	}

	if req.PIDs[0] != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting battle score record")
		return "", err
	}

	// make sure the player is not trying to submit a score for a battle that is expired
	isExpired, _ := db.GetBattleExpiryInfo(req.BattleID)
	if isExpired {
		log.Println("Battle", req.BattleID, "is expired, rejecting battle score record")
		return "", err
	}

	scoresCollection := database.Collection("scores")

	scoreHigher := []bool{}
	currentScore := []int{}

	for _, pid := range req.PIDs {
		var score models.Score
		score.OwnerPID = pid
		score.BattleID = req.BattleID
		score.Score = req.Score

		// Retrieve the existing score
		var existingScore models.Score
		err := scoresCollection.FindOne(context.TODO(), bson.M{"battle_id": req.BattleID, "pid": score.OwnerPID}).Decode(&existingScore)

		isNewScoreHigher := err == mongo.ErrNoDocuments || score.Score > existingScore.Score
		scoreHigher = append(scoreHigher, isNewScoreHigher)

		// Only update if the new score is higher
		if isNewScoreHigher {
			_, err = scoresCollection.UpdateOne(
				nil,
				bson.M{"battle_id": req.BattleID, "pid": score.OwnerPID},
				bson.D{
					{"$set", bson.D{
						{"battle_id", score.BattleID},
						{"pid", score.OwnerPID},
						{"score", score.Score},
					}},
				},
				options.Update().SetUpsert(true),
			)

			currentScore = append(currentScore, score.Score)
		} else {
			currentScore = append(currentScore, existingScore.Score)
		}
	}

	res := []BattleScoreRecordResponse{}

	numPids := len(req.PIDs)

	for i := 0; i < (numPids / 2); i++ {
		playerScoreIdx, _ := scoresCollection.CountDocuments(context.TODO(), bson.M{"battle_id": req.BattleID, "score": bson.M{"$gt": req.Score}})

		// Find the next highest score
		var nextHighestScore models.Score
		err = scoresCollection.FindOne(context.TODO(), bson.M{
			"battle_id": req.BattleID,
			"score":     bson.M{"$gt": req.Score},
		}, options.FindOne().SetSort(bson.D{{"score", 1}})).Decode(&nextHighestScore)

		if scoreHigher[i] {
			instaRankString := "b"

			// Get the name of the player who has the next highest score
			var name string = db.GetUsernameForPID(nextHighestScore.OwnerPID)
			var nextScoreDiff int
			if err == mongo.ErrNoDocuments {
				name = "N/A"
				nextScoreDiff = 0 // No higher score exists
			} else {
				name = db.GetUsernameForPID(nextHighestScore.OwnerPID)
				nextScoreDiff = nextHighestScore.Score - req.Score
			}

			if nextScoreDiff < 2000 && nextScoreDiff > 0 {
				instaRankString = "i|" + strconv.Itoa(nextScoreDiff) + "|" + name
			}

			instarank := BattleScoreRecordResponse{
				req.BattleID,
				1,
				int(playerScoreIdx + 1),
				0,
				"b",
				instaRankString,
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

	for i := numPids / 2; i < numPids; i++ {
		playerScoreIdx, _ := scoresCollection.CountDocuments(context.TODO(), bson.M{"battle_id": req.BattleID, "score": bson.M{"$gt": req.Score}})

		// Find the next highest score
		var nextHighestScore models.Score
		err = scoresCollection.FindOne(context.TODO(), bson.M{
			"battle_id": req.BattleID,
			"score":     bson.M{"$gt": req.Score},
		}, options.FindOne().SetSort(bson.D{{"score", 1}})).Decode(&nextHighestScore)

		if scoreHigher[i] {
			instaRankString := "b"

			// Get the name of the player who has the next highest score
			var name string = db.GetUsernameForPID(nextHighestScore.OwnerPID)
			var nextScoreDiff int
			if err == mongo.ErrNoDocuments {
				name = "N/A"
				nextScoreDiff = 0 // No higher score exists
			} else {
				name = db.GetUsernameForPID(nextHighestScore.OwnerPID)
				nextScoreDiff = nextHighestScore.Score - req.Score
			}

			if nextScoreDiff < 2000 && nextScoreDiff > 0 {
				instaRankString = "i|" + strconv.Itoa(nextScoreDiff) + "|" + name
			}

			instarank := BattleScoreRecordResponse{
				req.BattleID,
				0,
				int(playerScoreIdx + 1),
				0,
				"b",
				instaRankString,
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

	return marshaler.MarshalResponse(service.Path(), res)
}
