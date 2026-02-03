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

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)

	if !res {
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
		// check if the gathering was created by this machine (via created_by_machine_id field)
		// or if the creator username is a Master User from this machine
		machineID := db.GetMachineIDFromUsername(dbGathering.Creator)
		machineOwned := (dbGathering.CreatedByMachineID != 0 && dbGathering.CreatedByMachineID == client.MachineID()) ||
			(machineID != 0 && machineID == client.MachineID())

		if !machineOwned {
			log.Printf("Client %s is not the creator of gathering %v\n", client.Username, gatheringID)
			SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.NotAuthenticated)
			return
		}
	}

	// the client sends the entire gathering again, so update it in the DB
	// Only clear created_by_machine_id if the client is NOT a Master User
	// (i.e., they're logged into a real account and now own the gathering)

	updateDoc := bson.D{
		{"$set", bson.D{
			{"contents", gathering},
			{"public", g.HarmonixGathering.Public},
			{"last_updated", time.Now().Unix()},
			{"creator", client.Username},
		}},
	}

	// Only unset created_by_machine_id if the client is a regular user, not a Master User
	if !db.IsPIDAMasterUser(int(client.PlayerID())) {
		updateDoc = append(updateDoc, bson.E{"$unset", bson.D{{"created_by_machine_id", ""}}})
	}

	result, err := gatherings.UpdateOne(
		context.TODO(),
		bson.M{"gathering_id": gatheringID},
		updateDoc,
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
