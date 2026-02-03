package servers

import (
	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func Participate(err error, client *nex.Client, callID uint32, gatheringID uint32) {

	var res bool
	if client.Platform() == 2 {
		res, _ = ValidateClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)
	} else {
		res, _ = ValidateNonMasterClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)
	}

	if !res {
		return
	}

	rmcResponseStream := nex.NewStream()

	// i am not 100% sure what this method is for exactly
	rmcResponseStream.WriteUInt32LE(1) // response code

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
