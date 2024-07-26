package servers

import (
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

	// attempt to get a random gathering and deserialize it
	// any gatherings that havent been updated in 5 minutes are ignored
	// this should prevent endless loops of trying to join old/stale gatherings that are still in the DB
	// but any UI state change or playing a song will update the gathering
	cur, err := gatheringCollection.Aggregate(nil, []bson.M{
		bson.M{"$match": bson.D{
			// dont find our own gathering
			{
				Key:   "creator",
				Value: bson.D{{Key: "$ne", Value: client.Username}},
			},
			// only look for gatherings updated in the last 5 minutes
			{
				Key:   "last_updated",
				Value: bson.D{{Key: "$gt", Value: (time.Now().Unix()) - (5 * 60)}},
			},
			// dont look for gatherings in the "in song" state
			{
				Key:   "state",
				Value: bson.D{{Key: "$ne", Value: 2}},
			},
			// dont look for gatherings in the "on song select" state
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
			{
				Key:   "console_type",
				Value: bson.D{{Key: "$eq", Value: client.Platform()}},
			},
		}},
		bson.M{"$sample": bson.M{"size": 10}},
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
			rmcResponseStream.WriteU32LENext([]uint32{uint32(len(gathering.Contents) + 4)})
			rmcResponseStream.WriteU32LENext([]uint32{uint32(len(gathering.Contents))})
			rmcResponseStream.Grow(int64(len(gathering.Contents)))
			rmcResponseStream.WriteBytesNext(gathering.Contents[0:4])
			rmcResponseStream.WriteU32LENext([]uint32{user.PID})
			rmcResponseStream.WriteU32LENext([]uint32{user.PID})
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
