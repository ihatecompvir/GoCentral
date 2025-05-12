package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func GetConsoleUsernames(err error, client *nex.Client, callID uint32, friendCode string) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.NintendoManagementProtocolID)

	if !res {
		return
	}

	log.Printf("Getting console usernames for machine with friend code %v\n", friendCode)

	machinesCollection := database.GocentralDatabase.Collection("machines")

	// look up the machine with the associated friend code in the database
	var machine models.Machine

	_ = machinesCollection.FindOne(context.TODO(), bson.M{"wii_friend_code": friendCode}).Decode(&machine)

	var users []models.User

	// if the machine ID is 0, it doesn't exist
	if machine.MachineID == 0 {
		// TODO: fake machine ID we can use for Wii Friends that don't have RB3
		log.Printf("Machine with friend code %v does not exist\n", friendCode)
	} else {
		// add the master user to the usernames list
		var masterUser models.User
		masterUser.Username = "Master User (" + friendCode + ")"
		masterUser.PID = uint32(machine.MachineID)
		users = append(users, masterUser)

		// now that we have the machine ID, we can look up all associated users
		usersCollection := database.GocentralDatabase.Collection("users")

		cur, err := usersCollection.Find(context.TODO(), bson.M{"created_by_machine_id": machine.MachineID})

		if err != nil {
			log.Printf("Could not find users for machine %v: %v\n", machine.MachineID, err)
			SendErrorCode(SecureServer, client, nexproto.NintendoManagementProtocolID, callID, quazal.UnknownError)
			return
		}

		// iterate through the users and add them to the list
		for cur.Next(context.Background()) {
			var user models.User
			err := cur.Decode(&user)
			if err != nil {
				log.Printf("Could not decode user: %v\n", err)
				SendErrorCode(SecureServer, client, nexproto.NintendoManagementProtocolID, callID, quazal.UnknownError)
				return
			}
			// limit the amount of users reported to 4 (including the Master User)
			if len(users) <= 4 {
				users = append(users, user)
			}
		}
	}

	// create a stream to hold the response
	rmcResponseStream := nex.NewStream()

	rmcResponseStream.WriteUInt32LE(uint32(len(users)))

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
