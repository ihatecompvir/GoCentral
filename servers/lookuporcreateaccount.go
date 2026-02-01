package servers

import (
	"context"
	"fmt"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"rb3server/utils"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

// also handles Xbox 360 account switching
func LookupOrCreateAccount(err error, client *nex.Client, callID uint32, username string, key string, groups uint32, email string) {

	// don't allow users who have not logged in as a master user to create/lookup accounts
	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.AccountManagementProtocolID)

	if !res {
		return
	}

	rmcResponseStream := nex.NewStream()

	users := database.GocentralDatabase.Collection("users")
	machinesCollection := database.GocentralDatabase.Collection("machines")

	var user models.User

	if result := users.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user); result != nil {
		log.Printf("%s has never connected before - create DB entry\n", username)

		guid, err := generateGUID()

		// get the next PID atomically to avoid race conditions when running multiple ionstances
		newPID, err := database.GetNextPID(context.TODO())
		if err != nil {
			log.Printf("Could not get next PID: %s\n", err)
			SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
			return
		}

		if client.Platform() == 2 {
			_, err = users.InsertOne(context.TODO(), bson.D{
				{Key: "username", Value: username},
				{Key: "pid", Value: newPID},
				{Key: "console_type", Value: client.Platform()},
				{Key: "guid", Value: guid},
				{Key: "created_by_machine_id", Value: client.MachineID()},
			})
		} else {
			_, err = users.InsertOne(context.TODO(), bson.D{
				{Key: "username", Value: username},
				{Key: "pid", Value: newPID},
				{Key: "console_type", Value: client.Platform()},
				{Key: "guid", Value: guid},
			})
		}

		if err != nil {
			log.Printf("Could not create user %s: %s\n", username, err)
			SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
			return
		}

		// make sure we actually set the server-assigned PID to the new one when it is created
		client.SetPlayerID(user.PID)

		if err = users.FindOne(context.TODO(), bson.M{"username": username}).Decode(&user); err != nil {

			if err != nil {
				log.Printf("Could not find newly created user: %s\n", err)
				SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.OperationError)
				return
			}
		}
	}

	log.Printf("%s requesting to lookup or create an account\n", username)

	client.Username = username

	var stationURL string = "prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(client.ConnectionID())

	client.SetExternalStationURL(stationURL)
	client.SetPlayerID(user.PID)
	utils.GetClientStoreSingleton().AddClient(client.Address().String())
	utils.GetClientStoreSingleton().PushPID(client.Address().String(), client.PlayerID())

	if client.Platform() == 2 {
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
	}

	rmcResponseStream.WriteUInt32LE(user.PID)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.AccountManagementProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.LookupOrCreateAccount, rmcResponseBody)

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
