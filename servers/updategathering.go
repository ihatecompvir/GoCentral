package servers

import (
	"log"
	"rb3server/database"

	serialization "rb3server/serialization/gathering"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func UpdateGathering(err error, client *nex.Client, callID uint32, gathering []byte, gatheringID uint32) {

	var deserializer serialization.GatheringDeserializer

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to update a gathering without a valid server-assigned PID, rejecting call")
		return
	}

	g, err := deserializer.Deserialize(gathering)
	if err != nil {
		log.Printf("Could not deserialize the gathering!")
		return
	}

	if client.Username == "Master User" {
		log.Printf("Ignoring UpdateGathering for unauthenticated %s\n", client.WiiFC)
		return
	}
	log.Printf("Updating gathering for %s\n", client.Username)

	gatherings := database.GocentralDatabase.Collection("gatherings")

	// the client sends the entire gathering again, so update it in the DB

	result, err := gatherings.UpdateOne(
		nil,
		bson.M{"gathering_id": gatheringID},
		bson.D{
			{"$set", bson.D{{"contents", gathering}}},
			{"$set", bson.D{{"public", g.HarmonixGathering.Public}}},
			{"$set", bson.D{{"last_updated", time.Now().Unix()}}},
		},
	)

	if err != nil {
		log.Println("Could not update gathering for " + client.Username)
		return
	}

	log.Printf("Updated %v gatherings\n", result.ModifiedCount)

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.Grow(4)

	rmcResponseStream.WriteUInt8(1)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.UpdateGathering, rmcResponseBody)

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
