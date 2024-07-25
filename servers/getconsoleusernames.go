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

func GetConsoleUsernames(err error, client *nex.Client, callID uint32, friendCode string) {

	if client.MachineID() == 0 {
		log.Println("Client is trying to get console usernames without a valid server-assigned machine ID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.NintendoManagementProtocolID, callID, 0x00010001)
		return
	}

	log.Printf("Getting console usernames for machine with friend code %v\n", friendCode)

	machinesCollection := database.GocentralDatabase.Collection("machines")

	// look up the machine with the associated friend code in the database
	var machine models.Machine

	_ = machinesCollection.FindOne(context.TODO(), bson.M{"wii_friend_code": friendCode}).Decode(&machine)

	// if the machine ID is 0, it doesn't exist
	if machine.MachineID == 0 {
		log.Printf("Machine with friend code %v does not exist\n", friendCode)
		SendErrorCode(SecureServer, client, nexproto.NintendoManagementProtocolID, callID, 0x00010001)
		return
	}

	// now that we have the machine ID, we can look up all associated users
	usersCollection := database.GocentralDatabase.Collection("users")

	var users []models.User

	cur, err := usersCollection.Find(context.TODO(), bson.M{"created_by_machine_id": machine.MachineID})

	if err != nil {
		log.Printf("Could not find users for machine %v: %v\n", machine.MachineID, err)
		SendErrorCode(SecureServer, client, nexproto.NintendoManagementProtocolID, callID, 0x00010001)
		return
	}

	// iterate through the users and add them to the list
	for cur.Next(context.Background()) {
		var user models.User
		err := cur.Decode(&user)
		if err != nil {
			log.Printf("Could not decode user: %v\n", err)
			SendErrorCode(SecureServer, client, nexproto.NintendoManagementProtocolID, callID, 0x00010001)
			return
		}
		users = append(users, user)
	}

	// create a stream to hold the response
	rmcResponseStream := nex.NewStream()
	rmcResponseStream.Grow(10)

	rmcResponseStream.WriteU32LENext([]uint32{uint32(len(users) + 1)})

	rmcResponseStream.WriteBufferString("Master User (" + friendCode + ")")

	// write the usernames to the stream
	for _, user := range users {
		rmcResponseStream.WriteBufferString(user.Username)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.NintendoManagementProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.GetConsoleUsernames, rmcResponseBody)

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
