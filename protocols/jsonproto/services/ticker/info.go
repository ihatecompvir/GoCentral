package ticker

import (
	"context"
	"log"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/utils"
	"sync"
	"time"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	db "rb3server/database"
)

var (
	globalRankCache  map[int]int
	globalRankMu     sync.RWMutex
	globalRankExpiry time.Time

	roleRankCache  map[int]map[int]int // roleID -> (PID -> rank)
	roleRankMu     sync.RWMutex
	roleRankExpiry map[int]time.Time

	battleCountCache  int64
	battleCountMu     sync.RWMutex
	battleCountExpiry time.Time

	tickerCacheTTL = 5 * time.Minute
)

func getCachedGlobalRank(ctx context.Context, scoresCollection *mongo.Collection, pid int) (int, error) {
	globalRankMu.RLock()
	if globalRankCache != nil && time.Now().Before(globalRankExpiry) {
		rank, ok := globalRankCache[pid]
		total := len(globalRankCache)
		globalRankMu.RUnlock()
		if ok {
			return rank, nil
		}
		return total + 1, nil
	}
	globalRankMu.RUnlock()

	pipeline := mongo.Pipeline{
		{{"$group", bson.D{{"_id", "$pid"}, {"totalScore", bson.D{{"$sum", "$score"}}}}}},
		{{"$sort", bson.D{{"totalScore", -1}}}},
	}

	cursor, err := scoresCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID         int `bson:"_id"`
		TotalScore int `bson:"totalScore"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return 0, err
	}

	newCache := make(map[int]int, len(results))
	for i, result := range results {
		newCache[result.ID] = i + 1
	}

	globalRankMu.Lock()
	globalRankCache = newCache
	globalRankExpiry = time.Now().Add(tickerCacheTTL)
	globalRankMu.Unlock()

	if rank, ok := newCache[pid]; ok {
		return rank, nil
	}
	return len(newCache) + 1, nil
}

func getCachedRoleRank(ctx context.Context, scoresCollection *mongo.Collection, roleID int, pid int) (int, error) {
	roleRankMu.RLock()
	if roleRankCache != nil {
		if expiry, ok := roleRankExpiry[roleID]; ok && time.Now().Before(expiry) {
			if ranks, ok := roleRankCache[roleID]; ok {
				rank, found := ranks[pid]
				total := len(ranks)
				roleRankMu.RUnlock()
				if found {
					return rank, nil
				}
				return total + 1, nil
			}
		}
	}
	roleRankMu.RUnlock()

	rolePipeline := mongo.Pipeline{
		{{"$match", bson.D{{"role_id", roleID}}}},
		{{"$group", bson.D{{"_id", "$pid"}, {"totalScore", bson.D{{"$sum", "$score"}}}}}},
		{{"$sort", bson.D{{"totalScore", -1}}}},
	}

	cursor, err := scoresCollection.Aggregate(ctx, rolePipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID         int `bson:"_id"`
		TotalScore int `bson:"totalScore"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return 0, err
	}

	newRanks := make(map[int]int, len(results))
	for i, result := range results {
		newRanks[result.ID] = i + 1
	}

	roleRankMu.Lock()
	if roleRankCache == nil {
		roleRankCache = make(map[int]map[int]int)
		roleRankExpiry = make(map[int]time.Time)
	}
	roleRankCache[roleID] = newRanks
	roleRankExpiry[roleID] = time.Now().Add(tickerCacheTTL)
	roleRankMu.Unlock()

	if rank, ok := newRanks[pid]; ok {
		return rank, nil
	}
	return len(newRanks) + 1, nil
}

func getCachedBattleCount(ctx context.Context, setlistsCollection *mongo.Collection) (int64, error) {
	battleCountMu.RLock()
	if time.Now().Before(battleCountExpiry) {
		count := battleCountCache
		battleCountMu.RUnlock()
		return count, nil
	}
	battleCountMu.RUnlock()

	count, err := setlistsCollection.CountDocuments(ctx, bson.M{"type": bson.M{"$in": []int{1000, 1001, 1002}}})
	if err != nil {
		return 0, err
	}

	battleCountMu.Lock()
	battleCountCache = count
	battleCountExpiry = time.Now().Add(tickerCacheTTL)
	battleCountMu.Unlock()

	return count, nil
}

type TickerInfoRequest struct {
	Region      string `json:"region"`
	Locale      string `json:"locale"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	RoleID      int    `json:"role_id"` // current instrument?
}

type TickerInfoResponse struct {
	PID              int    `json:"pid"`
	MOTD             string `json:"motd"`
	BattleCount      int    `json:"battle_count"`
	RoleID           int    `json:"role_id"`
	RoleRank         int    `json:"role_rank"`
	RoleIsGlobal     int    `json:"role_is_global"`
	RoleIsPercentile int    `json:"role_is_percentile"`
	BandID           int    `json:"band_id"`
	BandRank         int    `json:"band_rank"`
	BankIsGlobal     int    `json:"band_is_global"`
	BandIsPercentile int    `json:"band_is_percentile"`
}

type TickerInfoService struct {
}

func (service TickerInfoService) Path() string {
	return "ticker/info/get"
}

func (service TickerInfoService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req TickerInfoRequest
	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	validPIDres, _ := utils.GetClientStoreSingleton().IsValidPID(client.Address().String(), uint32(req.PID))

	if !validPIDres {
		log.Println("Client is attempting to get ticker stats without a valid server-assigned PID, rejecting call")
		return "", nil
	}

	bandsCollection := database.Collection("bands")
	var band models.Band
	err = bandsCollection.FindOne(nil, bson.M{"pid": req.PID}).Decode(&band)

	ctx := context.TODO()

	battleCount, err := getCachedBattleCount(ctx, database.Collection("setlists"))
	if err != nil {
		return "", err
	}

	scoresCollection := database.Collection("scores")

	totalScoreRank, err := getCachedGlobalRank(ctx, scoresCollection, req.PID)
	if err != nil {
		return "", err
	}

	roleRank, err := getCachedRoleRank(ctx, scoresCollection, req.RoleID, req.PID)
	if err != nil {
		return "", err
	}

	res := []TickerInfoResponse{{
		req.PID,
		db.GetCoolFact(),
		int(battleCount),
		req.RoleID,
		roleRank,
		1,
		0,
		band.BandID,
		totalScoreRank,
		1,
		0,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
