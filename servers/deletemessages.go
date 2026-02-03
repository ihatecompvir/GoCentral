package servers

import (
	"log"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func DeleteMessages(err error, client *nex.Client, callID uint32, pid uint32, recipientType uint32, messageIDs []uint32) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.MessagingProtocolID)

	if !res {
		return
	}

	log.Printf("DeleteMessages for PID %d, deleting %d messages\n", pid, len(messageIDs))

	// Delete messages from the in-memory store
	GlobalMessageStore.DeleteMessages(pid, messageIDs)

	rmcResponseStream := nex.NewStream()
	// No response data for DeleteMessages

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MessagingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.DeleteMessages, rmcResponseBody)

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
