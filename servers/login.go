package servers

import (
	"crypto/hmac"
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"rb3server/database"
	"rb3server/models"
	"regexp"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

var machineType int = 255 // 0 = xbox, 1 = ps3, 2 = wii
func deriveKerberosKey(userPID uint32, pwd string) []byte {
	var kerberosTicketKey []byte

	// hardcoded dummy pwd, only guest doesn't use this password
	if pwd == "" {
		kerberosTicketKey = []byte(os.Getenv("USERPASSWORD"))
	} else {
		kerberosTicketKey = []byte(pwd)
	}

	for i := 0; i < 65000+(int(userPID)%1024); i++ {
		kerberosTicketKey = nex.MD5Hash(kerberosTicketKey)
	}

	return kerberosTicketKey
}

func generateKerberosTicket(userPID uint32, serverPID uint32, keySize int, pwd string) ([]byte, []byte) {

	sessionKey := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10}

	// Create ticket body info
	kerberosTicketInfoKey := make([]byte, 16)
	//rand.Read(kerberosTicketInfoKey) // TODO: enable random keys and make them shared with secure server

	ticketInfoEncryption := nex.NewKerberosEncryption(kerberosTicketInfoKey)
	ticketInfoStream := nex.NewStream()

	encryptedTicketInfo := ticketInfoEncryption.Encrypt(ticketInfoStream.Bytes())

	// Create ticket
	kerberosTicketKey := deriveKerberosKey(userPID, pwd)

	ticketEncryption := nex.NewKerberosEncryption(kerberosTicketKey)
	ticketStream := nex.NewStream()
	ticketStream.Grow(int64(24))

	ticketStream.WriteBytesNext(sessionKey)
	ticketStream.WriteU32LENext([]uint32{1})
	ticketStream.WriteU32LENext([]uint32{0x24})
	ticketStream.WriteBuffer(encryptedTicketInfo)
	return ticketEncryption.Encrypt(ticketStream.Bytes()), kerberosTicketKey
}

func Login(err error, client *nex.Client, callID uint32, username string) {
	serverPID := 2 // Quazal Rendez-Vous

	users := database.GocentralDatabase.Collection("users")
	configCollection := database.GocentralDatabase.Collection("config")

	var user models.User

	// check for Wii FC inside parentheses
	// TODO: - do this better, this feels bleh
	var rgx = regexp.MustCompile(`\(([^()]*)\)`)
	res := rgx.FindStringSubmatch(username)

	
	// If there is no regex found, we are a PS3 client so get the correct stuff from the DB for the user
	// PS3 usernames cannot contain parentheses so there is no chance of a PS3 client taking the wii path

	// (TODO) Add support for RPCS3 & Dolphin (Xenia can be added once they make networking on it better.)

	if client.Server().AccessKey() == "d52d1e000328fbc724fde65006b88b56" { // xbox 360
		log.Println("Xbox client connecting")
		machineType = 0
		client.Username = "XBOX"
	} else if client.Server().AccessKey() == "bfa620c57c2d3bcdf4362a6fa6418e58" {
		log.Println("PS3 client connecting")
		machineType = 1
		client.Username = "PS3"
	} else if client.Server().AccessKey() == "e97dc2ce9904698f84cae429a41b328a" {
		log.Println("Wii client connecting")
		machineType = 2
	} else {
		log.Println("Unknown machine connecting --- ABORT") // Basically it doesn't fall into this category 
		SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, 0x00010001)
		return
	}

	if machineType == 0 || machineType == 1 {
		if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
			log.Printf("%s has never connected before - create DB entry\n", username)
			_, err := users.InsertOne(nil, bson.D{
				{Key: "username", Value: username},
				{Key: "pid", Value: Config.LastPID + 1},
				{Key: "console_type", Value: machineType},
			})

			if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
				log.Printf("Could not find newly-created user %s: %s\n", username, err)
				SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, 0x00010001)
				return
			}

			_, err = configCollection.UpdateOne(
				nil,
				bson.M{},
				bson.D{
					{"$set", bson.D{{"last_pid", Config.LastPID + 1}}},
				},
			)
			if err != nil {
				log.Println("Could not update config in database: ", err)
			}

			Config.LastPID++

		}
		// client.Username = username
	} else if machineType == 2 {
		client.Username = "Master User"
		user.PID = 12345678 // master user PID is currently 12345678 - probably should go with 0 or something since it is a special account
		client.WiiFC = res[1]
		log.Printf("Wii client detected, friend code %v\n", client.WiiFC)
	}

	log.Printf("%s requesting log in, has PID %v\n", username, user.PID)
	var encryptedTicket []byte
	var kerberosKey []byte

	client.SetPlayerID(user.PID)

	// generate the ticket and pass the friend code as the pwd on Wii, or use static password on PS3
	if client.Username == "Master User" {
		encryptedTicket, kerberosKey = generateKerberosTicket(user.PID, uint32(serverPID), 16, client.WiiFC)
	} else {
		encryptedTicket, kerberosKey = generateKerberosTicket(user.PID, uint32(serverPID), 16, "")
	}
	mac := hmac.New(md5.New, kerberosKey)
	mac.Write(encryptedTicket)
	calculatedHmac := mac.Sum(nil)

	// Build the response body

	addr := os.Getenv("ADDRESS")

	if addr == "" {
		log.Println("ADDRESS is not set, clients will be unable to connect to the secure server. Please set the ADDRESS environment variable and restart GoCentral")
		SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, 0x00010001)
		return
	}

	stationURL := fmt.Sprintf("prudps:/address=%s;port=%s;CID=1;PID=2;sid=1;stream=3;type=2", os.Getenv("ADDRESS"), os.Getenv("SECUREPORT"))

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.Grow(int64(23))

	rmcResponseStream.WriteU32LENext([]uint32{0x10001}) // success
	rmcResponseStream.WriteU32LENext([]uint32{user.PID})
	rmcResponseStream.WriteBuffer(append(encryptedTicket[:], calculatedHmac[:]...))

	// RVConnectionData
	rmcResponseStream.WriteBufferString(stationURL) // Station
	rmcResponseStream.WriteU32LENext(([]uint32{0}))

	// dunno what this is looks like the response code again? not sure if its needed either but its on the end of real RB packets
	rmcResponseStream.WriteU32LENext([]uint32{0x1})
	rmcResponseStream.WriteU32LENext([]uint32{0x100})

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(nexproto.AuthenticationProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.AuthenticationMethodLogin, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	responsePacket, _ := nex.NewPacketV0(client, nil)

	responsePacket.SetVersion(0)
	responsePacket.SetSource(0x31)
	responsePacket.SetDestination(0x3F)
	responsePacket.SetType(nex.DataPacket)

	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	AuthServer.Send(responsePacket)

}
