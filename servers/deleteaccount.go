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

func DeleteAccount(err error, client *nex.Client, callID uint32, pid uint32) {

	if client.MachineID() == 0 {
		log.Println("Client is attempting to delete account without valid server-assigned Machine ID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, 0x00010001)
		return
	}

	usersCollection := database.GocentralDatabase.Collection("users")

	// get the user
	var user models.User
	if err = usersCollection.FindOne(context.TODO(),
		bson.M{"pid": pid}).Decode(&user); err != nil {
		log.Printf("Could not find user with PID %d: %+v\n", pid, err)
		SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, 0x00010001)
		return
	}

	// make sure the machine ID who created the user matches the one trying to delete it
	if user.CreatedByMachineID != client.MachineID() {
		log.Printf("Client with machine ID %d is trying to delete account with PID %d, but it was created by machine ID %d\n", client.MachineID(), pid, user.CreatedByMachineID)
		SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, 0x00010001)
		return
	}

	// delete the user
	if _, err = usersCollection.DeleteOne(nil, bson.M{"pid": pid}); err != nil {
		log.Printf("Could not delete user with PID %d: %+v\n", pid, err)
		SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, 0x00010001)
		return
	}

	// just respond with nothing for now
	rmcResponseStream := nex.NewStream()

	rmcResponseStream.Grow(10)

	rmcResponseStream.WriteU32LENext([]uint32{0})

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.AccountManagementProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.DeleteAccount, rmcResponseBody)

	responsePacket, _ := nex.NewPacketV0(client, nil)

	responsePacket.SetVersion(0)
	responsePacket.SetSource(0x31)
	responsePacket.SetDestination(0x3F)
	responsePacket.SetType(nex.DataPacket)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)
}
