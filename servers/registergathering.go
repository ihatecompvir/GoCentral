package servers

import (
	"log"
	"math/rand"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"

	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func RegisterGathering(err error, client *nex.Client, callID uint32, gathering []byte) {

	res, _ := ValidateNonMasterClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)

	if !res {
		return
	}

	log.Println("Registering gathering...")

	// delete old gatherings, and create a new gathering

	gatherings := database.GocentralDatabase.Collection("gatherings")

	gatheringID := rand.Intn(250000-500) + 500

	// Attempt to clear stale gatherings that may exist
	// If there are stale gatherings registered, other clients will try to connect to sessions that don't exist anymore
	deleteResult, deleteError := gatherings.DeleteMany(nil, bson.D{
		{Key: "creator", Value: client.Username},
	})

	if deleteError != nil {
		log.Println("Could not clear stale gatherings")
	}

	if deleteResult.DeletedCount != 0 {
		log.Printf("Successfully cleared %v stale gatherings for %s...\n", deleteResult.DeletedCount, client.Username)
	}

	// get the user
	users := database.GocentralDatabase.Collection("users")
	var user models.User
	err = users.FindOne(nil, bson.M{"username": client.Username}).Decode(&user)

	if err != nil {
		log.Println("User " + client.Username + " did not exist in database, could not register gathering")
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
		return
	}

	consoleType := client.Platform()

	if user.CrossplayEnabled == true {
		consoleType = 1
	}

	// Create a new gathering
	_, err = gatherings.InsertOne(nil, bson.D{
		{Key: "gathering_id", Value: gatheringID},
		{Key: "contents", Value: gathering},
		{Key: "creator", Value: client.Username},
		{Key: "last_updated", Value: time.Now().Unix()},
		{Key: "state", Value: 0},
		{Key: "public", Value: 0},
		{Key: "matchmaking_pool", Value: consoleType},
	})

	if err != nil {
		log.Printf("Failed to create gathering: %+v\n", err)
	}

	rmcResponseStream := nex.NewStream()

	rmcResponseStream.WriteUInt32LE(uint32(gatheringID)) // client expects the new gathering ID in the response

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.RegisterGathering, rmcResponseBody)

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
