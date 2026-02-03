package servers

import (
	"log"
	"rb3server/serialization/message"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func RetrieveMessages(err error, client *nex.Client, callID uint32, pid uint32, recipientType uint32, messageIDs []uint32, leaveOnServer bool) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.MessagingProtocolID)

	if !res {
		return
	}

	log.Printf("RetrieveMessages for PID %d, retrieving %d messages (leaveOnServer: %v)\n", pid, len(messageIDs), leaveOnServer)

	// Retrieve messages from the in-memory store
	// If leaveOnServer is false, delete them after retrieval
	messages := GlobalMessageStore.GetMessagesByIDs(pid, messageIDs, !leaveOnServer)

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.WriteUInt32LE(uint32(len(messages)))

	// Serialize each message as full TextMessage with AnyDataHolder wrapper
	serializer := &message.TextMessageDeserializer{}
	for _, msg := range messages {

		serializedMsg := serializer.Serialize(msg)
		rmcResponseStream.Grow(int64(len(serializedMsg)))
		rmcResponseStream.WriteBytesNext(serializedMsg)

		// debug print all fields of the message
		log.Printf("Retrieved Message ID %d from machine %s (PID %d) to recipient %d (type %d), Subject: %s, Body: %s\n",
			msg.ID, msg.Sender, msg.SenderPID, msg.IDRecipient, msg.RecipientType, msg.Subject, msg.TextBody)
	}

	log.Printf("Returning %d full messages for PID %d\n", len(messages), pid)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MessagingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.RetrieveMessages, rmcResponseBody)

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
