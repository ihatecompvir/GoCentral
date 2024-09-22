package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func CustomFind(err error, client *nex.Client, callID uint32, data []byte) {

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to check for gatherings without a valid server-assigned PID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.NotAuthenticated)
		return
	}

	if client.Username == "Master User" {
		log.Printf("Ignoring CheckForGatherings for unauthenticated Wii master user with friend code %s\n", client.WiiFC)
		SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.NotAuthenticated)
		return
	}
	log.Printf("Checking for available gatherings for %s...\n", client.Username)

	gatheringCollection := database.GocentralDatabase.Collection("gatherings")
	usersCollection := database.GocentralDatabase.Collection("users")

	// get user
	var user models.User
	if err = usersCollection.FindOne(nil, bson.M{"username": client.Username}).Decode(&user); err != nil {
		log.Printf("User %s does not exist in database, could not check for gatherings: %s\n", client.Username, err)
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
				Key: "matchmaking_pool",
				Value: bson.D{{
					Key: "$in",
					Value: func() []int {
						switch client.Platform() {
						case 2:
							if user.CrossplayEnabled {
								return []int{1, 3}
							} else {
								return []int{2}
							}
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
	for cur.Next(nil) {
		var g models.Gathering
		err = cur.Decode(&g)
		if err != nil {
			log.Printf("Error decoding gathering: %+v\n", err)
			SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.OperationError)
			return
		}
		gatherings = append(gatherings, g)
	}

	rmcResponseStream := nex.NewStream()

	// if there are no availble gatherings, tell the client to check again.
	// otherwise, pass the available gathering to the client
	if len(gatherings) == 0 {
		log.Println("There are no active gatherings. Tell client to keep checking")
		rmcResponseStream.WriteUInt32LE(0)
	} else {
		log.Printf("Found %d gatherings - telling client to attempt joining", len(gatherings))
		rmcResponseStream.WriteUInt32LE(uint32(len(gatherings)))
		for _, gathering := range gatherings {
			var user models.User

			if err = usersCollection.FindOne(nil, bson.M{"username": gathering.Creator}).Decode(&user); err != nil {
				log.Printf("Could not find creator %s of gathering: %+v\n", gathering.Creator, err)
				SendErrorCode(SecureServer, client, nexproto.CustomMatchmakingProtocolID, callID, quazal.OperationError)
				return
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
