package servers

import (
	"fmt"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func RequestURLs(err error, client *nex.Client, callID uint32, stationCID uint32, stationPID uint32) {
	rmcResponseStream := nex.NewStream()

	log.Printf("Requesting station URL for %v\n", stationPID)

	users := database.GocentralDatabase.Collection("users")

	var user models.User

	if err = users.FindOne(nil, bson.M{"pid": stationPID}).Decode(&user); err != nil {
		log.Println("Could not find user with PID " + fmt.Sprint(stationPID) + " in database")
		SendErrorCode(SecureServer, client, nexproto.SecureProtocolID, callID, quazal.InvalidPID)
		return
	}

	if user.IntStationURL != "" {
		rmcResponseStream.WriteUInt8(1)                         // response code
		rmcResponseStream.WriteUInt32LE(2)                      // the number of station urls present
		rmcResponseStream.WriteBufferString(user.StationURL)    // WAN station URL
		rmcResponseStream.WriteBufferString(user.IntStationURL) // LAN station URL used for connecting to other players on the same LAN
	} else {
		rmcResponseStream.WriteUInt8(1)                      // response code
		rmcResponseStream.WriteUInt32LE(1)                   // the number of station urls present
		rmcResponseStream.WriteBufferString(user.StationURL) // WAN station URL
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.SecureProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SecureMethodRequestURLs, rmcResponseBody)

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
