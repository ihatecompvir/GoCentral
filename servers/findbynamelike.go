package servers

import (
	"context"
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

func FindByNameLike(err error, client *nex.Client, callID uint32, uiGroups uint32, name string) {
	users := database.GocentralDatabase.Collection("users")
	var user models.User

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.AccountManagementProtocolID)

	if !res {
		return
	}

	log.Printf("Finding user by name like %s\n", name)

	// lookup the user by name
	if err = users.FindOne(context.TODO(), bson.M{"username": name}).Decode(&user); err != nil {
		var rgx = regexp.MustCompile(`\(([^()]*)\)`)
		res := rgx.FindStringSubmatch(name)

		if len(res) != 0 {
			machines := database.GocentralDatabase.Collection("machines")
			var machine models.Machine

			if err = machines.FindOne(context.TODO(), bson.M{"wii_friend_code": res[1]}).Decode(&machine); err != nil {
				log.Println("Could not find machine with friend code " + fmt.Sprint(res[1]) + " in database")
				SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.UnknownError)
				return
			}

			user.Username = "Master User (" + machine.WiiFriendCode + ")"
			user.PID = uint32(machine.MachineID)
		} else {
			log.Println("Could not find user or machine with name " + fmt.Sprint(name) + " in database")
			SendErrorCode(SecureServer, client, nexproto.AccountManagementProtocolID, callID, quazal.UnknownError)
			return
		}
	}

	rmcResponseStream := nex.NewStream()

	rmcResponseStream.WriteUInt32LE(1)

	rmcResponseStream.WriteUInt32LE(user.PID)
	rmcResponseStream.WriteBufferString(user.Username)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.AccountManagementProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.FindByNameLike, rmcResponseBody)

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
