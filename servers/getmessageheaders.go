package servers

import (
	"log"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func GetMessageHeaders(err error, client *nex.Client, callID uint32, pid uint32, gatheringID uint32, rangeOffset uint32, rangeSize uint32) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.MessagingProtocolID)

	if !res {
		return
	}

	log.Printf("Getting message headers for PID %v\n", pid)
	rmcResponseStream := nex.NewStream()
	rmcResponseStream.WriteUInt32LE(0)
	rmcResponseStream.WriteUInt32LE(0)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MessagingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.GetMessageHeaders, rmcResponseBody)

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
