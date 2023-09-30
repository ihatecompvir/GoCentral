package servers

import (
	"log"

	"github.com/knvihatecompvirtva/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func Participate(err error, client *nex.Client, callID uint32, gatheringID uint32) {

	if client.PlayerID() == 0 {
		log.Println("Client is participate in a gathering without a valid server-assigned PID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, 0x00010001)
		return
	}

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.Grow(4)

	// i am not 100% sure what this method is for exactly
	rmcResponseStream.WriteUInt8(1) // response code

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.Participate, rmcResponseBody)

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
