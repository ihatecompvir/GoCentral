package servers

import (
	"context"
	"log"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"

	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func SetState(err error, client *nex.Client, callID uint32, gatheringID uint32, state uint32) {

	res, _ := ValidateNonMasterClientPID(SecureServer, client, callID, nexproto.MatchmakingProtocolID)

	if !res {
		return
	}

	log.Printf("Setting state to %v for gathering %v...\n", state, gatheringID)

	rmcResponseStream := nex.NewStream()

	gatherings := database.GocentralDatabase.Collection("gatherings")
	var gathering models.Gathering
	err = gatherings.FindOne(nil, bson.M{"gathering_id": gatheringID}).Decode(&gathering)

	if err != nil {
		log.Printf("Could not find gathering %v to set the state on: %v\n", gatheringID, err)
		SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
		return
	} else {
		// Update the gathering.Contents, State, and LastUpdated fields
		gathering.Contents[0x1C] = (byte)(state>>(8*0)) & 0xff
		gathering.Contents[0x1D] = (byte)(state>>(8*1)) & 0xff
		gathering.Contents[0x1E] = (byte)(state>>(8*2)) & 0xff
		gathering.Contents[0x1F] = (byte)(state>>(8*3)) & 0xff

		update := bson.M{
			"$set": bson.M{
				"contents":     gathering.Contents,
				"state":        state,
				"last_updated": time.Now().Unix(),
			},
		}

		_, err = gatherings.UpdateOne(context.TODO(), bson.M{"gathering_id": gatheringID}, update)
		if err != nil {
			log.Printf("Could not set state for gathering %v: %v\n", gatheringID, err)
			SendErrorCode(SecureServer, client, nexproto.MatchmakingProtocolID, callID, quazal.OperationError)
			return
		} else {
			rmcResponseStream.WriteUInt8(1)
		}
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SetState, rmcResponseBody)

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
