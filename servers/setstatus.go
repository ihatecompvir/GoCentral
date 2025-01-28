package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func SetStatus(err error, client *nex.Client, callID uint32, status string) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.AccountManagementProtocolID)

	if !res {
		return
	}

	machinesCollection := database.GocentralDatabase.Collection("machines")
	_, err = machinesCollection.UpdateOne(
		context.TODO(),
		bson.M{"machine_id": client.PlayerID()},
		bson.D{
			{"$set", bson.D{{"status", status}}},
		},
	)

	if err != nil {
		log.Printf("Could not update status for machine %s: %s\n", client.Username, err)
		SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
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
