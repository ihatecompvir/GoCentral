package servers

import (
	"log"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func GetMessageHeaders(err error, client *nex.Client, callID uint32, pid uint32, recipientType uint32, rangeOffset uint32, rangeSize uint32) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.MessagingProtocolID)

	if !res {
		return
	}

	if recipientType == 1 {
		log.Printf("Getting message headers for PID %v\n", pid)
	} else {
		// honestly not sure if this can ever even happen but keeping it for completeness
		log.Printf("Getting message headers for Gathering ID %v\n", pid)
	}

	// Retrieve messages from the in-memory store
	messages := GlobalMessageStore.GetMessages(pid)

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.WriteUInt32LE(uint32(len(messages)))

	// Write each UserMessage directly (no AnyDataHolder wrapper)
	for _, msg := range messages {
		rmcResponseStream.WriteUInt32LE(msg.ID)
		rmcResponseStream.WriteUInt32LE(msg.IDRecipient)
		rmcResponseStream.WriteUInt32LE(msg.RecipientType)
		rmcResponseStream.WriteUInt32LE(msg.ParentID)
		rmcResponseStream.WriteUInt32LE(msg.SenderPID)
		rmcResponseStream.WriteUInt64LE(msg.ReceptionTime.Value)
		rmcResponseStream.WriteUInt32LE(msg.LifeTime)
		rmcResponseStream.WriteUInt32LE(msg.Flags)
		rmcResponseStream.WriteBufferString(msg.Subject)
		rmcResponseStream.WriteBufferString(msg.Sender)
	}

	log.Printf("Returning %d messages for PID %v\n", len(messages), pid)

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
