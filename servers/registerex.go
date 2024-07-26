package servers

import (
	"fmt"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"regexp"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func RegisterEx(err error, client *nex.Client, callID uint32, stationUrls []string, className string, ticketData []byte) {
	users := database.GocentralDatabase.Collection("users")
	machines := database.GocentralDatabase.Collection("machines")

	var user models.User
	var machine models.Machine

	if err = users.FindOne(nil, bson.M{"username": client.Username}).Decode(&user); err != nil {
		err = machines.FindOne(nil, bson.M{"wii_friend_code": client.WiiFC}).Decode(&machine)

		if err != nil {
			log.Println("User or machine " + client.Username + " did not exist in database, could not register")
			SendErrorCode(SecureServer, client, nexproto.SecureProtocolID, callID, quazal.OperationError)
			return
		}

	}

	newRVCID := uint32(AuthServer.ConnectionIDCounter().Increment())

	// Build the response body
	rmcResponseStream := nex.NewStream()
	rmcResponseStream.Grow(200)

	rmcResponseStream.WriteU16LENext([]uint16{0x01})     // likely a response code of sorts
	rmcResponseStream.WriteU16LENext([]uint16{0x01})     // same as above
	rmcResponseStream.WriteU32LENext([]uint32{newRVCID}) // RVCID

	if user.PID != 0 {
		client.SetPlayerID(user.PID)
	} else {
		client.SetPlayerID(uint32(machine.MachineID))
	}

	// check if the PID is not the master PID. if it is the master PID, do not update the station URLs
	if user.PID != 12345678 && len(stationUrls) != 0 {

		var stationURL string = "prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(newRVCID)

		// run a RegEx to extract the IP address from the station URL
		re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)

		ipRegexResults := re.FindAllString(stationUrls[0], -1)
		var internalStationURL string

		// if there aren't any results, use a blank internal IP URL
		if len(ipRegexResults) != 0 {
			internalStationURL = "prudp:/address=" + ipRegexResults[0] + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(newRVCID)
		} else {
			internalStationURL = ""
			log.Printf("Client with PID %v did not have internal station URL, using empty string\n", user.PID)
		}

		// className is "XboxUserInfo" if the console is an Xbox
		// className is "NintendoToken" if the console is a Wii
		// className is "SonyNPTicket" if the console is a PS3

		consoleType := 0

		switch className {
		case "XboxUserInfo":
			consoleType = 0
		case "SonyNPTicket":
			consoleType = 1
		case "NintendoToken":
			consoleType = 2
		default:
			log.Println("Invalid ticket presented, could not determine console type")
			SendErrorCode(SecureServer, client, nexproto.SecureProtocolID, callID, quazal.InvalidArgument)
			return
		}

		// update station URLs and current console type
		result, err := users.UpdateOne(
			nil,
			bson.M{"username": client.Username},
			bson.D{
				{"$set", bson.D{{"station_url", stationURL}}},
				{"$set", bson.D{{"int_station_url", internalStationURL}}},
				{"$set", bson.D{{"console_type", consoleType}}},
			},
		)

		client.SetPlatform(consoleType)
		client.SetExternalStationURL(stationURL)
		client.SetConnectionID(uint32(newRVCID))

		if err != nil {
			log.Printf("Could not update station URLs for %s\n", result.ModifiedCount, client.Username)
			SendErrorCode(SecureServer, client, nexproto.SecureProtocolID, callID, quazal.OperationError)
			return
		}

		if result.ModifiedCount > 1 || result.ModifiedCount == 0 {
			log.Printf("Updated %v station URLs for %s \n", result.ModifiedCount, client.Username)
		} else {
			log.Printf("Updated %v station URL for %s \n", result.ModifiedCount, client.Username)
		}
	}

	// The game doesn't appear to do anything with this, but return something proper anyway
	rmcResponseStream.WriteBufferString("prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";sid=15;type=3")

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(nexproto.SecureProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SecureMethodRegisterEx, rmcResponseBody)

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
