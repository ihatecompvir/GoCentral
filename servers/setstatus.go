package servers

import (
	"log"
	"rb3server/database"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func SetStatus(err error, client *nex.Client, callID uint32, status string) {

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to update their status without a valid server-assigned PID, rejecting call")
	}
	usersCollection := database.GocentralDatabase.Collection("users")
	_, err = usersCollection.UpdateOne(
		nil,
		bson.M{"username": client.Username},
		bson.D{
			{"$set", bson.D{{"status", status}}},
		},
	)

	if err != nil {
		log.Printf("Could not update status for user %s: %s\n", client.Username, err)
		SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, 0x00010001)
		return
	}

	rmcResponseStream := nex.NewStream()

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.AccountManagementProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SetStatus, rmcResponseBody)

	responsePacket, _ := nex.NewPacketV0(client, nil)

	responsePacket.SetVersion(0)
	responsePacket.SetSource(0x31)
	responsePacket.SetDestination(0x3F)
	responsePacket.SetType(nex.DataPacket)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)

}
