package servers

import (
	"log"
	"rb3server/database"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func TerminateGathering(err error, client *nex.Client, callID uint32, gatheringID uint32) {

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to terminate a gathering without a valid server-assigned PID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, 0x00010001)
		return
	}

	if client.Username == "Master User" {
		log.Printf("Ignoring TerminateGathering for unauthenticated %s\n", client.WiiFC)
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, 0x00010001)
		return
	}
	log.Printf("Terminating gathering for %s...\n", client.Username)

	gatherings := database.RockcentralDatabase.Collection("gatherings")

	// remove the gathering from the DB so other players won't attempt to connect to it later
	result, err := gatherings.DeleteOne(
		nil,
		bson.M{"gathering_id": gatheringID},
	)

	if err != nil {
		log.Printf("Could not terminate gathering: %s\n", err)
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, 0x00010001)
		return
	}

	log.Printf("Terminated %v gathering\n", result.DeletedCount)

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.Grow(4)

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
