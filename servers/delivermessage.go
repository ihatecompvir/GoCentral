package servers

import (
	"log"
	"rb3server/serialization/message"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func DeliverMessage(err error, client *nex.Client, callID uint32, data []byte) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.MessageDeliveryProtocolID)

	if !res {
		return
	}

	// Deserialize the TextMessage
	deserializer := &message.TextMessageDeserializer{}
	msg, err := deserializer.Deserialize(data)
	if err != nil {
		log.Printf("Failed to deserialize TextMessage: %v\n", err)
	} else {
		// Store the message in the in-memory store for the recipient
		if GlobalMessageStore != nil {
			// Assign a unique incrementing message ID
			msg.ID = GlobalMessageStore.NextMessageID()

			// Set the sender's PID
			msg.SenderPID = client.PlayerID()

			// Set reception time to now
			msg.ReceptionTime = message.DateTimeNow()

			msg.Sender = "Master User (" + client.WiiFC + ")"

			log.Printf("DeliverMessage ID %d from machine %s (PID %d) to recipient %d (type %d)\n",
				msg.ID, msg.Sender, msg.SenderPID, msg.IDRecipient, msg.RecipientType)

			GlobalMessageStore.AddMessage(msg.IDRecipient, msg)
		}
	}

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.WriteUInt32LE(0) // there is no response data according to NintendoClients wiki, so just return a null response

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MessageDeliveryProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.DeliverMessage, rmcResponseBody)

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
