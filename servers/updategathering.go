package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"

	serialization "rb3server/serialization/gathering"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"

	db "rb3server/database"
)

func UpdateGathering(err error, client *nex.Client, callID uint32, gathering []byte, gatheringID uint32) {

	var deserializer serialization.GatheringDeserializer

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to update a gathering without a valid server-assigned PID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.NotAuthenticated)
		return
	}

	g, err := deserializer.Deserialize(gathering)
	if err != nil {
		log.Printf("Could not deserialize the gathering!")
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	log.Printf("Updating gathering ID %v for %s\n", gatheringID, client.Username)

	gatherings := database.GocentralDatabase.Collection("gatherings")

	// get and deserialize the gathering from the DB
	var dbGathering models.Gathering

	err = gatherings.FindOne(context.TODO(), bson.M{"gathering_id": gatheringID}).Decode(&dbGathering)

	if err != nil {
		log.Println("Could not find gathering ID for " + client.Username)
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	// make sure the client's PID matches the creator of the gathering
	if dbGathering.Creator != db.GetUsernameForPID(int(client.PlayerID())) {
		// check if the creator of the gathering was created by the machine's master user
		machineID := db.GetMachineIDFromUsername(dbGathering.Creator)

		if machineID != client.MachineID() || machineID == 0 {
			log.Printf("Client %s is not the creator of gathering %v\n", client.Username, gatheringID)
			SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.NotAuthenticated)
			return
		}
	}

	// the client sends the entire gathering again, so update it in the DB

	result, err := gatherings.UpdateOne(
		context.TODO(),
		bson.M{"gathering_id": gatheringID},
		bson.D{
			{"$set", bson.D{{"contents", gathering}}},
			{"$set", bson.D{{"public", g.HarmonixGathering.Public}}},
			{"$set", bson.D{{"last_updated", time.Now().Unix()}}},
			{"$set", bson.D{{"creator", client.Username}}},
		},
	)

	if err != nil {
		log.Println("Could not update gathering for " + client.Username)
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	log.Printf("Updated %v gatherings\n", result.ModifiedCount)

	rmcResponseStream := nex.NewStream()

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
