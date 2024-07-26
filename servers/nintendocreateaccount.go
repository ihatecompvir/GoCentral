package servers

import (
	"context"
	"fmt"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

// also handles Xbox 360 account switching
func NintendoCreateAccount(err error, client *nex.Client, callID uint32, username string, key string, groups uint32, email string) {

	rmcResponseStream := nex.NewStream()

	users := database.GocentralDatabase.Collection("users")
	configCollection := database.GocentralDatabase.Collection("config")
	machinesCollection := database.GocentralDatabase.Collection("machines")

	var user models.User

	var config models.Config
	err = configCollection.FindOne(context.TODO(), bson.M{}).Decode(&config)
	if err != nil {
		log.Printf("Could not get config %v\n", err)
	}

	if result := users.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user); result != nil {
		log.Printf("%s has never connected before - create DB entry\n", username)

		guid, err := generateGUID()

		_, err = users.InsertOne(context.TODO(), bson.D{
			{Key: "username", Value: username},
			{Key: "pid", Value: config.LastPID + 1},
			{Key: "console_type", Value: client.Platform()},
			{Key: "guid", Value: guid},
			{Key: "created_by_machine_id", Value: client.MachineID()},
		})

		if err != nil {
			log.Printf("Could not create Nintendo user %s: %s\n", username, err)
			SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
			return
		}

		_, err = configCollection.UpdateOne(
			context.TODO(),
			bson.M{},
			bson.D{
				{"$set", bson.D{{"last_pid", config.LastPID + 1}}},
			},
		)
		if err != nil {
			log.Println("Could not update config in database: ", err)
			SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
			return
		}

		Config.LastPID = config.LastPID + 1
		Config.LastMachineID = config.LastMachineID
		Config.LastBandID = config.LastBandID
		Config.LastSetlistID = config.LastSetlistID
		Config.LastCharacterID = config.LastCharacterID

		// make sure we actually set the server-assigned PID to the new one when it is created
		client.SetPlayerID(user.PID)

		if err = users.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user); err != nil {

			if err != nil {
				log.Printf("Could not find newly created Nintendo user: %s\n", err)
				SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
				return
			}
		}
	}

	log.Printf("%s requesting Nintendo log in from Wii Friend Code %s, has PID %v\n", username, client.WiiFC, user.PID)

	client.Username = username

	var stationURL string = "prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(client.ConnectionID())

	client.SetExternalStationURL(stationURL)
	client.SetPlayerID(user.PID)

	// update station URL of the machine that created the user
	result, err := machinesCollection.UpdateOne(
		context.TODO(),
		bson.M{"machine_id": client.MachineID()},
		bson.D{
			{"$set", bson.D{{"station_url", stationURL}}},
		},
	)

	if err != nil {
		log.Printf("Could not update station URLs for machine ID %v: %s\n", client.MachineID(), err)
		SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
		return
	}

	log.Printf("Updated %v station URL for machine ID %v \n", result.ModifiedCount, client.MachineID())

	rmcResponseStream.WriteUInt32LE(user.PID)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.AccountManagementProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.NintendoCreateAccount, rmcResponseBody)

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
