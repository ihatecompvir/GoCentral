package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func TerminateGathering(err error, client *nex.Client, callID uint32, gatheringID uint32) {

	var res bool
	if client.Platform() == 2 {
		res, _ = ValidateClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)
	} else {
		res, _ = ValidateNonMasterClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)
	}

	if !res {
		return
	}

	log.Printf("Terminating gathering ID %v for %s...\n", gatheringID, client.Username)

	gatherings := database.GocentralDatabase.Collection("gatherings")

	// Verify ownership before deleting
	var dbGathering models.Gathering
	err = gatherings.FindOne(context.TODO(), bson.M{"gathering_id": gatheringID}).Decode(&dbGathering)
	if err != nil {
		log.Printf("Could not find gathering %v to terminate: %s\n", gatheringID, err)
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	// Check ownership: either creator matches, or machine owns it
	if dbGathering.Creator != database.GetUsernameForPID(int(client.PlayerID())) {
		machineID := database.GetMachineIDFromUsername(dbGathering.Creator)
		machineOwned := (dbGathering.CreatedByMachineID != 0 && dbGathering.CreatedByMachineID == client.MachineID()) ||
			(machineID != 0 && machineID == client.MachineID())

		if !machineOwned {
			log.Printf("Client %s is not the creator of gathering %v\n", client.Username, gatheringID)
			SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.NotAuthenticated)
			return
		}
	}

	// remove the gathering from the DB so other players won't attempt to connect to it later
	result, err := gatherings.DeleteOne(
		context.TODO(),
		bson.M{"gathering_id": gatheringID},
	)

	if err != nil {
		log.Printf("Could not terminate gathering: %s\n", err)
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	log.Printf("Terminated %v gathering\n", result.DeletedCount)

	rmcResponseStream := nex.NewStream()

	rmcResponseStream.WriteUInt8(1)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.TerminateGathering, rmcResponseBody)

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
