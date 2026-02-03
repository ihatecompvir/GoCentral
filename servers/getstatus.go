package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/models"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func GetStatus(err error, client *nex.Client, callID uint32, pid uint32) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.AccountManagementProtocolID)

	if !res {
		return
	}

	var status string

	log.Printf("Getting status for PID %d\n", pid)

	machines := database.GocentralDatabase.Collection("machines")
	var machine models.Machine

	if err = machines.FindOne(context.TODO(), bson.M{"machine_id": pid}).Decode(&machine); err != nil {
		log.Printf("Could not find machine with PID %d in database\n", pid)
		status = "Offline"
	} else {
		status = machine.Status
	}

	log.Printf("Responding %s\n", status)

	rmcResponseStream := nex.NewStream()

	rmcResponseStream.WriteBufferString(status)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.AccountManagementProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.GetStatus, rmcResponseBody)

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
