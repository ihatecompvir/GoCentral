package servers

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"regexp"

	"rb3server/authentication"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

	rmcResponseStream.WriteUInt16LE(0x01)     // likely a response code of sorts
	rmcResponseStream.WriteUInt16LE(0x01)     // same as above
	rmcResponseStream.WriteUInt32LE(newRVCID) // RVCID

	if user.PID != 0 {
		client.SetPlayerID(user.PID)
	} else {
		client.SetPlayerID(uint32(machine.MachineID))
	}

	if len(stationUrls) != 0 {

		var stationURL string = "prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(newRVCID)

		// run a RegEx to extract the IP address from the station URL
		re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)

		ipRegexResults := re.FindAllString(stationUrls[0], -1)
		var internalStationURL string

		// if there aren't any results, use a blank internal IP URL
		if len(ipRegexResults) != 0 {
			internalStationURL = "prudp:/address=" + ipRegexResults[0] + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;RVCID=" + fmt.Sprint(newRVCID)
		} else {
			internalStationURL = ""
			log.Printf("Client with PID %v did not have internal station URL, using empty string\n", user.PID)
		}

		// className is "XboxUserInfo" if the console is an Xbox
		// className is "NintendoToken" if the console is a Wii
		// className is "SonyNPTicket" if the console is a PS3
		// className is "RPCN" if the emulator is RPCS3

		consoleType := 0

		var ticketDataToEncode []byte

		switch className {
		case "XboxUserInfo":
			consoleType = 0
		case "SonyNPTicket":
			ticketDataToEncode = ticketData[8:]
			rpcn := []byte("RPCN")
			isRPCN := bytes.Contains(ticketData, rpcn)

			if isRPCN {
				consoleType = 3
			} else {
				consoleType = 1
			}
		case "NintendoToken":
			ticketDataToEncode = ticketData
			consoleType = 2

		default:
			log.Println("Invalid ticket presented, could not determine console type")
			SendErrorCode(SecureServer, client, nexproto.SecureProtocolID, callID, quazal.InvalidArgument)
			return
		}

		if os.Getenv("TICKETVERIFIERENDPOINT") != "" {
			ticketVerifier := &authentication.TicketVerifier{TicketVerifierEndpoint: os.Getenv("TICKETVERIFIERENDPOINT")}

			if !ticketVerifier.VerifyTicket(ticketDataToEncode, consoleType) {
				log.Println("Invalid ticket presented, could not verify ticket")

				// reject the client and then reset their stuff
				SendErrorCode(SecureServer, client, nexproto.SecureProtocolID, callID, quazal.AccessDenied)
				client.Reset()
				SecureServer.Kick(client)
				client.SetPlayerID(0)
				return
			}
		}

		// update station URLs and current console type

		var result *mongo.UpdateResult = nil

		if client.PlayerID() != uint32(machine.MachineID) {
			result, _ = users.UpdateOne(
				nil,
				bson.M{"username": client.Username},
				bson.D{
					{"$set", bson.D{{"station_url", stationURL}}},
					{"$set", bson.D{{"int_station_url", internalStationURL}}},
					{"$set", bson.D{{"console_type", consoleType}}},
				},
			)
		} else {
			result, _ = machines.UpdateOne(
				nil,
				bson.M{"machine_id": machine.MachineID},
				bson.D{
					{"$set", bson.D{{"station_url", stationURL}}},
				},
			)
		}

		client.SetPlatform(consoleType)
		client.SetExternalStationURL(stationURL)
		client.SetConnectionID(uint32(newRVCID))

		if err != nil {
			log.Printf("Could not update station URLs for %s\n", result.ModifiedCount, client.Username)
			SendErrorCode(SecureServer, client, nexproto.SecureProtocolID, callID, quazal.OperationError)
			return
		}

		if client.PlayerID() != uint32(machine.MachineID) {
			if result.ModifiedCount > 1 || result.ModifiedCount == 0 {
				log.Printf("Updated %v station URLs for %s \n", result.ModifiedCount, client.Username)
			} else {
				log.Printf("Updated %v station URL for %s \n", result.ModifiedCount, client.Username)
			}
		} else {
			if result.ModifiedCount > 1 || result.ModifiedCount == 0 {
				log.Printf("Updated %v station URLs for machine ID %v \n", result.ModifiedCount, client.MachineID())
			} else {
				log.Printf("Updated %v station URL for machine ID %v \n", result.ModifiedCount, client.MachineID())
			}
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
