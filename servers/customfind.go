package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"sort"
	"strings"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func CustomFind(err error, client *nex.Client, callID uint32, data []byte) {

	res, _ := ValidateNonMasterClientPID(SecureServer, client, callID, nexproto.CustomMatchmakingProtocolID)

	if !res {
		return
	}

	log.Printf("Checking for available gatherings for %s...\n", client.Username)

	gatheringCollection := database.GocentralDatabase.Collection("gatherings")
	usersCollection := database.GocentralDatabase.Collection("users")

	// Fetch the searching user to get their USIDs
	var searchingUser models.User
	err = usersCollection.FindOne(context.TODO(), bson.M{"pid": client.PlayerID()}).Decode(&searchingUser)
	if err != nil {
		log.Printf("Could not find searching user %s: %v\n", client.Username, err)
		SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	// attempt to get a random gathering and deserialize it
	// any gatherings that havent been updated in 5 minutes are ignored
	// this should prevent endless loops of trying to join old/stale gatherings that are still in the DB
	// but any UI state change or playing a song will update the gathering
	cur, err := gatheringCollection.Aggregate(context.TODO(), []bson.M{
		{"$match": bson.D{
			// don't find our own gathering
			{
				Key:   "creator",
				Value: bson.D{{Key: "$ne", Value: client.Username}},
			},
			// only look for gatherings updated in the last 5 minutes
			{
				Key:   "last_updated",
				Value: bson.D{{Key: "$gt", Value: (time.Now().Unix()) - (5 * 60)}},
			},
			// don't look for gatherings in the "in song" state
			{
				Key:   "state",
				Value: bson.D{{Key: "$ne", Value: 2}},
			},
			// don't look for gatherings in the "on song select" state
			{
				Key:   "state",
				Value: bson.D{{Key: "$ne", Value: 6}},
			},
			// only look for public gatherings
			{
				Key:   "public",
				Value: bson.D{{Key: "$eq", Value: 1}},
			},
			// only look for gatherings created by the current console type
			// with an additional match so RPCN can join real PS3 h/w gatherings
			{
				Key: "console_type",
				Value: bson.D{{
					Key: "$in",
					Value: func() []int {
						switch client.Platform() {
						case 1, 3:
							return []int{1, 3}
						default:
							return []int{client.Platform()}
						}
					}(),
				}},
			},
		}},
		{"$sample": bson.M{"size": 10}},
	})
	if err != nil {
		log.Printf("Could not get a random gathering: %s\n", err)
		SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.OperationError)
		return
	}
	var gatherings = make([]models.Gathering, 0)
	var creatorNames []string
	for cur.Next(nil) {
		var g models.Gathering
		err = cur.Decode(&g)
		if err != nil {
			log.Printf("Error decoding gathering: %+v\n", err)
			SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.OperationError)
			return
		}
		gatherings = append(gatherings, g)
		creatorNames = append(creatorNames, g.Creator)
	}

	// Fetch all creators in one go
	var creators []models.User
	if len(creatorNames) == 0 {
		// No gatherings found, skip straight to reporting no results
		rmcResponseStream := nex.NewStream()
		log.Println("There are no active gatherings. Tell client to keep checking")
		rmcResponseStream.WriteUInt32LE(0)

		rmcResponseBody := rmcResponseStream.Bytes()
		rmcResponse := nex.NewRMCResponse(nexproto.CustomMatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.RegisterGathering, rmcResponseBody)
		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)
		responsePacket.SetPayload(rmcResponseBytes)
		responsePacket.AddFlag(nex.FlagNeedsAck)
		responsePacket.AddFlag(nex.FlagReliable)
		SecureServer.Send(responsePacket)
		return
	}
	creatorCursor, err := usersCollection.Find(context.TODO(), bson.M{"username": bson.M{"$in": creatorNames}})
	if err != nil {
		log.Printf("Could not fetch gathering creators: %v\n", err)
		SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.OperationError)
		return
	}
	if err = creatorCursor.All(context.TODO(), &creators); err != nil {
		log.Printf("Could not decode gathering creators: %v\n", err)
		SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	creatorsMap := make(map[string]models.User)
	for _, creator := range creators {
		creatorsMap[creator.Username] = creator
	}

	// parse searching user's songs
	searcherSongs := make(map[string]bool)
	for _, songID := range strings.Split(searchingUser.USIDs, ",") {
		if songID != "" {
			searcherSongs[strings.TrimSpace(songID)] = true
		}
	}

	// calculate a similarity score for each gathering
	type ScoredGathering struct {
		Gathering models.Gathering
		Score     int
	}
	var scoredGatherings []ScoredGathering

	for _, g := range gatherings {
		score := 0
		if creator, exists := creatorsMap[g.Creator]; exists {
			creatorSongs := strings.Split(creator.USIDs, ",")
			for _, songID := range creatorSongs {
				if searcherSongs[strings.TrimSpace(songID)] {
					score++
				}
			}
		}
		scoredGatherings = append(scoredGatherings, ScoredGathering{Gathering: g, Score: score})
	}

	// sort by score descending
	sort.Slice(scoredGatherings, func(i, j int) bool {
		return scoredGatherings[i].Score > scoredGatherings[j].Score
	})

	// take top 10
	var topGatherings []models.Gathering
	limit := 10
	if len(scoredGatherings) < limit {
		limit = len(scoredGatherings)
	}
	for i := 0; i < limit; i++ {
		topGatherings = append(topGatherings, scoredGatherings[i].Gathering)
	}

	rmcResponseStream := nex.NewStream()

	// if there are no availble gatherings, tell the client to check again.
	// otherwise, pass the available gathering to the client
	if len(topGatherings) == 0 {
		log.Println("There are no active gatherings. Tell client to keep checking")
		rmcResponseStream.WriteUInt32LE(0)
	} else {
		log.Printf("Found %d gatherings - telling client to attempt joining", len(topGatherings))
		rmcResponseStream.WriteUInt32LE(uint32(len(topGatherings)))
		for _, gathering := range topGatherings {
			// We already have the creator in our map, no need to query again
			user, exists := creatorsMap[gathering.Creator]
			if !exists {
				// Fallback just in case, though shouldn't happen given previous query
				if err = usersCollection.FindOne(nil, bson.M{"username": gathering.Creator}).Decode(&user); err != nil {
					log.Printf("Could not find creator %s of gathering: %+v\n", gathering.Creator, err)
					continue
				}
			}

			rmcResponseStream.WriteBufferString("HarmonixGathering")
			rmcResponseStream.WriteUInt32LE(uint32(len(gathering.Contents) + 4))
			rmcResponseStream.WriteUInt32LE(uint32(len(gathering.Contents)))
			rmcResponseStream.Grow(int64(len(gathering.Contents)))
			rmcResponseStream.WriteBytesNext(gathering.Contents[0:4])
			rmcResponseStream.WriteUInt32LE(user.PID)
			rmcResponseStream.WriteUInt32LE(user.PID)
			rmcResponseStream.WriteBytesNext(gathering.Contents[12:])
		}
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.CustomMatchmakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.RegisterGathering, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	responsePacket, _ := nex.NewPacketV0(client, nil)

	responsePacket.SetVersion(0)
	responsePacket.SetSource(0x31)
	responsePacket.SetDestination(0x3F)
	responsePacket.SetType(nex.DataPacket)

	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)

}
