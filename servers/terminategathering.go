package servers

import (
	"log"
	"rb3server/database"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func TerminateGathering(err error, client *nex.Client, callID uint32, gatheringID uint32) {

	res, _ := ValidateNonMasterClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)

	if !res {
		return
	}

	log.Printf("Terminating gathering ID %v for %s...\n", gatheringID, client.Username)

	gatherings := database.GocentralDatabase.Collection("gatherings")

	// remove the gathering from the DB so other players won't attempt to connect to it later
	result, err := gatherings.DeleteOne(
		nil,
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
