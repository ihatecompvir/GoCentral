package servers

import (
	"fmt"
	"log"
	"rb3server/database"
	"rb3server/models"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func FindBySingleID(err error, client *nex.Client, callID uint32, gatheringID uint32) {
	users := database.GocentralDatabase.Collection("users")
	gatheringCollection := database.GocentralDatabase.Collection("gatherings")
	var user models.User
	var gathering models.Gathering

	rmcResponseStream := nex.NewStream()

	if err = gatheringCollection.FindOne(nil, bson.M{"gathering_id": gatheringID}).Decode(&gathering); err != nil {
		log.Printf("Could not find gatheringID %s of gathering: %+v\n", gatheringID, err)
		rmcResponseStream.WriteUInt8(0)
		return
	} else {

		if err = users.FindOne(nil, bson.M{"username": gathering.Creator}).Decode(&user); err != nil {
			log.Println("Could not find user with username " + fmt.Sprint(gathering.Creator) + " in database")
			rmcResponseStream.WriteUInt8(0)
			return
		} else {
			rmcResponseStream.WriteUInt8(1)

			rmcResponseStream.WriteBufferString("HarmonixGathering")
			rmcResponseStream.WriteU32LENext([]uint32{uint32(len(gathering.Contents) + 4)})
			rmcResponseStream.WriteU32LENext([]uint32{uint32(len(gathering.Contents))})
			rmcResponseStream.Grow(int64(len(gathering.Contents)))
			rmcResponseStream.WriteBytesNext(gathering.Contents[0:4])
			rmcResponseStream.WriteU32LENext([]uint32{user.PID})
			rmcResponseStream.WriteU32LENext([]uint32{user.PID})
			rmcResponseStream.WriteBytesNext(gathering.Contents[12:])
		}
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.FindBySingleID, rmcResponseBody)

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
