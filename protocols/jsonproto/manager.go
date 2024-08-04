package jsonproto

import (
	"fmt"
	"log"
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
	"rb3server/protocols/jsonproto/services/misc"
	"rb3server/protocols/jsonproto/services/music_library"
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
	mgr.register(setlists.SetlistSyncService{})
	mgr.register(setlists.SetlistUpdateService{})

	// account linking
	mgr.register(accountlink.AccountLinkService{})

	// ticker for instaranks and etc.
	mgr.register(ticker.TickerInfoService{})

	// entities
	mgr.register(character.CharacterUpdateService{})
	mgr.register(band.BandUpdateService{})
	mgr.register(entities.GetLinkcodeService{})

	// performance
	mgr.register(performance.PerformanceRecordService{})

	// accomplishments
	mgr.register(accomplishment.AccomplishmentRecordService{})

	// leaderboards
	mgr.register(leaderboard.MaxrankGetService{})
	mgr.register(leaderboard.PlayerGetService{})
	mgr.register(leaderboard.AccPlayerGetService{})
	mgr.register(leaderboard.AccMaxrankGetService{})
	mgr.register(leaderboard.AccRankRangeGetService{})
	mgr.register(leaderboard.RankRangeGetService{})
	mgr.register(leaderboard.BattleMaxrankGetService{})
	mgr.register(leaderboard.BattlePlayerGetService{})
	mgr.register(leaderboard.BattleRankRangeGetService{})
	mgr.register(leaderboard.PlayerranksGetService{})
	mgr.register(leaderboard.FriendsUpdateService{})

	// songlists
	mgr.register(songlists.GetSonglistsService{})

	// battles
	mgr.register(battles.GetBattlesClosedService{})
	mgr.register(battles.LimitCheckService{})
	mgr.register(battles.BattleCreateService{})

	// score recording
	mgr.register(scores.ScoreRecordService{})
	mgr.register(scores.BattleScoreRecordService{})

	// stats
	mgr.register(stats.StatsPadService{})

	// misc
	mgr.register(misc.MiscSyncAvailableSongsService{})
	mgr.register(misc.SetlistCreationStatusService{})

	// music_library
	mgr.register(music_library.SortAndFiltersService{})

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
		log.Printf("Unimplemented JSON service for path: %s\n", methodPath)
		return "", fmt.Errorf("unimplemented service for path:%s\n", methodPath)
	}

	return service.Handle(jsonStr, database.GocentralDatabase, client)

}
