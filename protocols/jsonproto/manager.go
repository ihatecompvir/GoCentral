package jsonproto

import (
	"fmt"
	"rb3server/database"
	"rb3server/protocols/jsonproto/marshaler"
	"rb3server/protocols/jsonproto/services/accomplishment"
	"rb3server/protocols/jsonproto/services/accountlink"
	"rb3server/protocols/jsonproto/services/battles"
	"rb3server/protocols/jsonproto/services/config"
	"rb3server/protocols/jsonproto/services/entities"
	"rb3server/protocols/jsonproto/services/entities/band"
	"rb3server/protocols/jsonproto/services/entities/character"
	leaderboard "rb3server/protocols/jsonproto/services/leaderboards"
	"rb3server/protocols/jsonproto/services/performance"
	"rb3server/protocols/jsonproto/services/scores"
	"rb3server/protocols/jsonproto/services/setlists"
	"rb3server/protocols/jsonproto/services/songlists"
	"rb3server/protocols/jsonproto/services/stats"
	"rb3server/protocols/jsonproto/services/ticker"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type Service interface {
	// the unique path to lookup the service by
	Path() string

	// function to process request
	Handle(string, *mongo.Database, *nex.Client) (string, error)
}

type ServicesManager struct {
	services map[string]Service
}

// Creates new services manager
func NewServicesManager() *ServicesManager {
	mgr := &ServicesManager{
		services: make(map[string]Service),
	}

	// register all services
	mgr.registerAll()

	return mgr

}

// register all services
func (mgr *ServicesManager) registerAll() {
	// config
	mgr.register(config.ConfigService{})

	// setlist creation
	mgr.register(setlists.SetlistCreationService{})
	mgr.register(setlists.SetlistSyncService{})
	mgr.register(setlists.SetlistUpdateService{})

	mgr.register(accountlink.AccountLinkService{})

	mgr.register(scores.ScoreRecordService{})

	mgr.register(ticker.TickerInfoService{})

	mgr.register(character.CharacterUpdateService{})
	mgr.register(band.BandUpdateService{})
	mgr.register(entities.GetLinkcodeService{})

	mgr.register(performance.PerformanceRecordService{})

	mgr.register(accomplishment.AccomplishmentRecordService{})

	mgr.register(leaderboard.MaxrankGetService{})
	mgr.register(leaderboard.PlayerGetService{})
	mgr.register(leaderboard.AccPlayerGetService{})
	mgr.register(leaderboard.AccMaxrankGetService{})
	mgr.register(leaderboard.AccRankRangeGetService{})
	mgr.register(leaderboard.RankRangeGetService{})

	mgr.register(songlists.GetSonglistsService{})

	mgr.register(battles.GetBattlesService{})
	mgr.register(battles.LimitCheckService{})

	mgr.register(stats.StatsPadService{})

}

// register a single service
func (mgr *ServicesManager) register(service Service) {
	mgr.services[service.Path()] = service
}

// delegates the request to the proper service
func (mgr ServicesManager) Handle(jsonStr string, client *nex.Client) (string, error) {

	methodPath, err := marshaler.GetRequestName(jsonStr)
	if err != nil {
		return "", err
	}

	// check service is implemented
	service, exists := mgr.services[methodPath]
	if !exists {
		return "", fmt.Errorf("unimplemented service for path:%s\n", methodPath)
	}

	return service.Handle(jsonStr, database.GocentralDatabase, client)

}
