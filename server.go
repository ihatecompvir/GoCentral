package main

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rc4"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"rb3server/models"
	"rb3server/protocols/jsonproto"
	"regexp"
	"syscall"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var config models.Config

func main() {
	uri := os.Getenv("MONGOCONNECTIONSTRING")

	if uri == "" {
		log.Fatalln("GoCentral relies on MongoDB. You must set a MongoDB connection string to use GoCentral")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {
		log.Fatalln("Could not connect to MongoDB: ", err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatalln("Could not connect to MongoDB: ", err)
		}
	}()

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalln("Could not ping MongoDB: ", err)
	}

	log.Println("Successfully established connection to MongoDB")

	gocentralDatabase := client.Database("gocentral")

	configCollection := gocentralDatabase.Collection("config")

	// get config from DB
	err = configCollection.FindOne(nil, bson.M{}).Decode(&config)
	if err != nil {
		log.Println("Could not get config from MongoDB database, creating default config: ", err)
		_, err = configCollection.InsertOne(nil, bson.D{
			{Key: "last_pid", Value: 500},
		})

		config.LastPID = 500

		if err != nil {
			log.Fatalln("Could not create default config! GoCentral cannot proceed: ", err)
		}
	}

	go mainAuth(gocentralDatabase)
	go mainSecure(gocentralDatabase)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Printf("Signal (%s) received, stopping\n", s)
}

func mainAuth(database *mongo.Database) {

	nexServer := nex.NewServer()

	nexServer.SetPrudpVersion(0)
	nexServer.SetSignatureVersion(1)
	nexServer.SetKerberosKeySize(16)
	nexServer.SetChecksumVersion(1)
	nexServer.UsePacketCompression(false)
	nexServer.SetFlagsVersion(0)
	nexServer.SetAccessKey(os.Getenv("ACCESSKEY"))
	nexServer.SetFragmentSize(962)

	authenticationServer := nexproto.NewAuthenticationProtocol(nexServer)

	authenticationServer.Login(func(err error, client *nex.Client, callID uint32, username string) {
		serverPID := 2 // Quazal Rendez-Vous

		users := database.Collection("users")
		configCollection := database.Collection("config")

		var user models.User

		// check for Wii FC inside parentheses
		// TODO: - do this better, this feels bleh
		var rgx = regexp.MustCompile(`\(([^()]*)\)`)
		res := rgx.FindStringSubmatch(username)

		// If there is no regex found, we are a PS3 client so get the correct stuff from the DB for the user
		// PS3 usernames cannot contain parentheses so there is no chance of a PS3 client taking the wii path
		if len(res) == 0 {
			if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
				log.Printf("%s has never connected before - create DB entry\n", username)
				_, err := users.InsertOne(nil, bson.D{
					{Key: "username", Value: username},
					{Key: "pid", Value: config.LastPID + 1},
				})

				if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
					log.Printf("Could not find newly-created user %s: %s\n", username, err)
				}

				_, err = configCollection.UpdateOne(
					nil,
					bson.M{},
					bson.D{
						{"$set", bson.D{{"last_pid", config.LastPID + 1}}},
					},
				)
				if err != nil {
					log.Println("Could not update config in database: ", err)
				}

				config.LastPID++

			}
			client.Username = username
		} else {
			client.Username = "Master User"
			user.PID = 12345678 // master user PID is currently 12345678 - probably should go with 0 or something since it is a special account
			client.WiiFC = res[1]
			log.Printf("Wii client detected, friend code %v\n", client.WiiFC)
		}

		log.Printf("%s requesting log in, has PID %v\n", username, user.PID)
		var encryptedTicket []byte
		var kerberosKey []byte

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
			log.Println("ADDRESS is not set, clients will likely be unable to connect to the secure server. Please set the ADDRESS environment variable and restart GoCentral")
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

		// add one empty byte to each decrypted payload
		// nintendos rendez-vous doesn't require this so its not implemented by default
		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	authenticationServer.RequestTicket(func(err error, client *nex.Client, callID uint32, userPID uint32, serverPID uint32) {
		log.Printf("PID %v requesting ticket...\n", userPID)

		encryptedTicket, kerberosKey := generateKerberosTicket(userPID, uint32(serverPID), 16, client.WiiFC)
		mac := hmac.New(md5.New, kerberosKey)
		mac.Write(encryptedTicket)
		calculatedHmac := mac.Sum(nil)

		// Build the response body
		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(int64(4 + 4 + len(encryptedTicket) + 0x10))

		rmcResponseStream.WriteU32LENext([]uint32{0x10001}) // success
		rmcResponseStream.WriteBuffer(append(encryptedTicket[:], calculatedHmac[:]...))

		rmcResponseBody := rmcResponseStream.Bytes()

		// Build response packet
		rmcResponse := nex.NewRMCResponse(nexproto.AuthenticationProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.AuthenticationMethodRequestTicket, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		// add one empty byte to each decrypted payload
		// nintendos rendez-vous doesn't require this so its not implemented by default
		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	ip := os.Getenv("LISTENINGIP")
	authPort := os.Getenv("AUTHPORT")

	if ip == "" {
		log.Println("No listening IP specified for the auth server, you may experience issues connecting. Please set the LISTENINGIP environment variable.")
	}

	if authPort == "" {
		log.Fatalln("No auth server port specified, please set the AUTHPORT environment variable. The default auth server port for various platforms can be found at https://github.com/ihatecompvir/GoCentral/wiki/Game-specific-Network-Details")
	}

	nexServer.Listen(ip + ":" + authPort)

}

func mainSecure(database *mongo.Database) {
	nexServer := nex.NewServer()
	nexServer.SetPrudpVersion(0)
	nexServer.SetSignatureVersion(1)
	nexServer.SetKerberosKeySize(16)
	nexServer.SetChecksumVersion(1)
	nexServer.UsePacketCompression(false)
	nexServer.SetFlagsVersion(0)
	nexServer.SetAccessKey(os.Getenv("ACCESSKEY"))
	nexServer.SetFragmentSize(962)

	secureServer := nexproto.NewSecureProtocol(nexServer)
	jsonServer := nexproto.NewJsonProtocol(nexServer)
	matchmakingServer := nexproto.NewMatchmakingProtocol(nexServer)
	natTraversalServer := nexproto.NewNATTraversalProtocol(nexServer)
	accountManagementServer := nexproto.NewAccountManagementProtocol(nexServer)
	unknownProtocolServer := nexproto.NewUnknownProtocol(nexServer)

	// Handle PRUDP CONNECT packet (not an RMC method)
	nexServer.On("Connect", func(packet *nex.PacketV0) {
		packet.Sender().SetClientConnectionSignature(packet.Sender().ClientConnectionSignature())

		// Decrypt payload
		decryptedPayload := make([]byte, 0x100)
		packet.Sender().Decipher().XORKeyStream(decryptedPayload, packet.Payload())
		stream := nex.NewStreamIn(decryptedPayload, packet.Sender().Server())
		stream.Grow(0x48)

		// get the ticket data and such
		// skip past the kerberos ticket
		stream.ReadBytesNext(4)
		stream.ReadBytesNext(0x20)
		stream.ReadBytesNext(9)
		requestData := stream.ReadBytesNext(0x1c)

		// TODO: use random key from auth server
		sessionKey := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10}

		requestDataEncryption, _ := rc4.NewCipher(sessionKey)
		decryptedRequestData := make([]byte, 0x1C)
		requestDataEncryption.XORKeyStream(decryptedRequestData, requestData)
		requestDataStream := nex.NewStreamIn(decryptedRequestData, nexServer)

		// extract the PID
		userPid := requestDataStream.ReadU32LENext(1)[0]

		// Get username for client from PID. This avoids having to grab it from the ticket
		// On Wii, the ticket does not contain the username so this is a platform-agnostic solution
		var user models.User
		users := database.Collection("users")
		users.FindOne(nil, bson.M{"pid": userPid}).Decode(&user)
		packet.Sender().Username = user.Username

		_ = requestDataStream.ReadU32LENext(1)[0]
		responseCheck := requestDataStream.ReadU32LENext(1)[0]

		responseValueStream := nex.NewStreamIn(make([]byte, 20), nexServer)
		responseValueBufferStream := nex.NewStream()
		responseValueBufferStream.Grow(20)

		responseValueStream.WriteU32LENext([]uint32{responseCheck + 1})
		responseValueBufferStream.WriteBuffer(responseValueStream.Bytes())

		packet.Sender().UpdateRC4Key(sessionKey)

		responsePacket, _ := nex.NewPacketV0(packet.Sender(), nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.ConnectPacket)

		tmpBuffer := make([]byte, responseValueBufferStream.ByteCapacity()+1)
		copy(tmpBuffer[1:len(tmpBuffer)-1], responseValueBufferStream.Bytes()[0:responseValueBufferStream.ByteCapacity()])
		bytes := make([]byte, len(tmpBuffer))
		packet.Sender().Cipher().XORKeyStream(bytes, tmpBuffer)
		responsePacket.SetPayload(bytes)
		responsePacket.AddFlag(nex.FlagAck)

		nexServer.Send(responsePacket)
	})

	secureServer.RegisterEx(func(err error, client *nex.Client, callID uint32, stationUrls []string, className string, ticketData []byte) {

		users := database.Collection("users")

		var user models.User

		if err = users.FindOne(nil, bson.M{"username": client.Username}).Decode(&user); err != nil {
			log.Println("User " + client.Username + " did not exist in database, could not register")
			return
		}

		// Build the response body
		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(200)

		rmcResponseStream.WriteU16LENext([]uint16{0x01})     // likely a response code of sorts
		rmcResponseStream.WriteU16LENext([]uint16{0x01})     // same as above
		rmcResponseStream.WriteU32LENext([]uint32{user.PID}) // pid

		// the RVCID must differ across all clients otherwise clients will reject each other
		randomRVCID := rand.Intn(250000-500) + 500

		// check if the PID is not the master PID. if it is the master PID, do not update the station URLs
		if user.PID != 12345678 && len(stationUrls) != 0 {

			var stationURL string = "prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(randomRVCID)

			// run a RegEx to extract the IP address from the station URL
			re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)

			ipRegexResults := re.FindAllString(stationUrls[0], -1)
			var internalStationURL string

			// if there aren't any results, use a blank internal IP URL
			if len(ipRegexResults) != 0 {
				internalStationURL = "prudp:/address=" + ipRegexResults[0] + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(randomRVCID)
			} else {
				internalStationURL = ""
			}

			// update station URLs
			result, err := users.UpdateOne(
				nil,
				bson.M{"username": client.Username},
				bson.D{
					{"$set", bson.D{{"station_url", stationURL}}},
					{"$set", bson.D{{"int_station_url", internalStationURL}}},
				},
			)

			if err != nil {
				log.Fatalln(err)
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

		// add one empty byte to each decrypted payload
		// nintendos rendez-vous doesn't require this so its not implemented by default
		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	secureServer.RequestURLs(func(err error, client *nex.Client, callID uint32, stationCID uint32, stationPID uint32) {
		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		log.Printf("Requesting station URL for %v\n", stationPID)

		users := database.Collection("users")

		var user models.User

		if err = users.FindOne(nil, bson.M{"pid": stationPID}).Decode(&user); err != nil {
			log.Println("Could not find user with PID " + fmt.Sprint(stationPID) + " in database")
			return
		}

		rmcResponseStream.WriteUInt8(1)                         // response code
		rmcResponseStream.WriteU32LENext([]uint32{2})           // the number of station urls present
		rmcResponseStream.WriteBufferString(user.StationURL)    // WAN station URL
		rmcResponseStream.WriteBufferString(user.IntStationURL) // LAN station URL used for connecting to other players on the same LAN

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.SecureProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.SecureMethodRequestURLs, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	jsonMgr := jsonproto.NewServicesManager()
	jsonServer.JSONRequest(func(err error, client *nex.Client, callID uint32, rawJson string) {

		// the JSON server will handle the request depending on what needs to be returned
		res, err := jsonMgr.Handle(rawJson, database)
		if err != nil {
			//log.Printf("Failed to handle JSON request: %+v", err)
			res = "[]"
		}

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.WriteBufferString(res)

		rmcResponseBody := rmcResponseStream.Bytes()

		// Build response packet
		rmcResponse := nex.NewRMCResponse(nexproto.JsonProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.JsonRequest, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		// add one empty byte to each decrypted payload
		// nintendos rendez-vous doesn't require this so its not implemented by default
		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	jsonServer.JSONRequest2(func(err error, client *nex.Client, callID uint32, rawJson string) {

		// I believe the second request method never returns a payload

		rmcResponseStream := nex.NewStream()

		rmcResponseBody := rmcResponseStream.Bytes()

		// Build response packet
		rmcResponse := nex.NewRMCResponse(nexproto.JsonProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.JsonRequest2, rmcResponseBody)

		// Even though this is a JSON-style method, it returns an empty body unlike JSONRequest
		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		// add one empty byte to each decrypted payload
		// nintendos rendez-vous doesn't require this so its not implemented by default
		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	matchmakingServer.RegisterGathering(func(err error, client *nex.Client, callID uint32, gathering []byte) {
		if client.Username == "Master User" {
			log.Printf("Ignoring RegisterGathering for unauthenticated %s\n", client.WiiFC)
			return
		}
		log.Println("Registering gathering...")

		// delete old gatherings, and create a new gathering

		gatherings := database.Collection("gatherings")

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

		// Create a new gathering
		_, err = gatherings.InsertOne(nil, bson.D{
			{Key: "gathering_id", Value: gatheringID},
			{Key: "contents", Value: gathering},
			{Key: "creator", Value: client.Username},
		})

		if err != nil {
			log.Printf("Failed to create gathering: %+v\n", err)
		}

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteU32LENext([]uint32{uint32(gatheringID)}) // client expects the new gathering ID in the response

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.RegisterGathering, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	matchmakingServer.UpdateGathering(func(err error, client *nex.Client, callID uint32, gathering []byte, gatheringID uint32) {
		if client.Username == "Master User" {
			log.Printf("Ignoring UpdateGathering for unauthenticated %s\n", client.WiiFC)
			return
		}
		log.Printf("Updating gathering for %s\n", client.Username)

		gatherings := database.Collection("gatherings")

		// the client sends the entire gathering again, so update it in the DB

		result, err := gatherings.UpdateOne(
			nil,
			bson.M{"gathering_id": gatheringID},
			bson.D{
				{"$set", bson.D{{"contents", gathering}}},
			},
		)

		if err != nil {
			log.Println("Could not update gathering for " + client.Username)
			return
		}

		log.Printf("Updated %v gatherings\n", result.ModifiedCount)

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteU32LENext([]uint32{gatheringID})

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.UpdateGathering, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	matchmakingServer.Participate(func(err error, client *nex.Client, callID uint32, gatheringID uint32) {

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		// i am not 100% sure what this method is for exactly
		rmcResponseStream.WriteUInt8(1) // response code

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.Participate, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	matchmakingServer.Unparticipate(func(err error, client *nex.Client, callID uint32, gatheringID uint32) {

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		// i am not 100% sure what this method is for, but it is the inverse of participate
		rmcResponseStream.WriteUInt8(1)

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.Unparticipate, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	matchmakingServer.LaunchSession(func(err error, client *nex.Client, callID uint32, gatheringID uint32) {
		log.Printf("Launching session for %s...\n", client.Username)

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteUInt8(1)

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.LaunchSession, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	matchmakingServer.TerminateGathering(func(err error, client *nex.Client, callID uint32, gatheringID uint32) {
		if client.Username == "Master User" {
			log.Printf("Ignoring TerminateGathering for unauthenticated %s\n", client.WiiFC)
			return
		}
		log.Printf("Terminating gathering for %s...\n", client.Username)

		gatherings := database.Collection("gatherings")

		// remove the gathering from the DB so other players won't attempt to connect to it later
		result, err := gatherings.DeleteOne(
			nil,
			bson.M{"gathering_id": gatheringID},
		)

		if err != nil {
			log.Printf("Could not terminate gathering: %s\n", err)
			return
		}

		log.Printf("Terminated %v gathering\n", result.DeletedCount)

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteUInt8(1)

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.TerminateGathering, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	matchmakingServer.CheckForGatherings(func(err error, client *nex.Client, callID uint32, data []byte) {
		if client.Username == "Master User" {
			log.Printf("Ignoring CheckForGatherings for unauthenticated %s\n", client.WiiFC)
			return
		}
		log.Printf("Checking for available gatherings for %s...\n", client.Username)

		gatherings := database.Collection("gatherings")
		users := database.Collection("users")

		var gathering models.Gathering
		var user models.User

		// attempt to get a random gathering and deserialize it
		cur, err := gatherings.Aggregate(nil, []bson.M{
			bson.M{"$match": bson.D{{
				Key:   "creator",
				Value: bson.D{{Key: "$ne", Value: client.Username}},
			}}},
			bson.M{"$sample": bson.M{"size": 1}},
		})
		if err != nil {
			log.Printf("Could not get a random gathering: %s\n", err)
		}
		cur.Next(nil)
		cur.Decode(&gathering)

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(19)

		// if there are no availble gatherings, tell the client to check again.
		// otherwise, pass the available gathering to the client
		if len(gathering.Contents) == 0 {
			log.Println("There are no active gatherings. Tell client to keep checking")
			rmcResponseStream.WriteU32LENext([]uint32{0})
		} else {
			log.Println("Found a gathering - attempting join!")
			if err = users.FindOne(nil, bson.M{"username": gathering.Creator}).Decode(&user); err != nil {
				log.Printf("Could not find creator %s of gathering: %+v\n", gathering.Creator, err)
			}
			rmcResponseStream.WriteU32LENext([]uint32{1})
			rmcResponseStream.WriteBufferString("HarmonixGathering")
			rmcResponseStream.WriteU32LENext([]uint32{uint32(len(gathering.Contents) + 4)})
			rmcResponseStream.WriteU32LENext([]uint32{uint32(len(gathering.Contents))})
			rmcResponseStream.Grow(int64(len(gathering.Contents)))
			rmcResponseStream.WriteBytesNext(gathering.Contents[0:4])
			rmcResponseStream.WriteU32LENext([]uint32{user.PID})
			rmcResponseStream.WriteU32LENext([]uint32{user.PID})
			rmcResponseStream.WriteBytesNext(gathering.Contents[12:])
		}

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID2, callID)
		rmcResponse.SetSuccess(nexproto.RegisterGathering, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	matchmakingServer.SetState(func(err error, client *nex.Client, callID uint32, gatheringID uint32, state uint32) {
		log.Printf("Setting state for gathering %v...\n", gatheringID)

		rmcResponseStream := nex.NewStream()

		gatherings := database.Collection("gatherings")
		var gathering models.Gathering
		err = gatherings.FindOne(nil, bson.M{"gathering_id": gatheringID}).Decode(&gathering)

		if err != nil {
			log.Printf("Could not find gathering %v to set the state on: %v\n", gatheringID, err)
		}

		// TODO: Replace with something better
		gathering.Contents[0x1C] = (byte)(state>>(8*0)) & 0xff
		gathering.Contents[0x1D] = (byte)(state>>(8*1)) & 0xff
		gathering.Contents[0x1E] = (byte)(state>>(8*2)) & 0xff
		gathering.Contents[0x1F] = (byte)(state>>(8*3)) & 0xff

		_, err = gatherings.ReplaceOne(nil, bson.M{"gathering_id": gatheringID}, gathering)
		if err != nil {
			log.Printf("Could not set state for gathering %v: %v\n", gatheringID, err)
		}
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteUInt8(1)

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakingProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.SetState, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	natTraversalServer.InitiateProbe(func(err error, client *nex.Client, callID uint32, stationURL string) {
		log.Printf("Client is performing a NAT traversal probe to %s...\n", stationURL)

		rmcResponseStream := nex.NewStream()

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.NATTraversalID, callID)
		rmcResponse.SetSuccess(nexproto.InitiateProbe, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	accountManagementServer.NintendoCreateAccount(func(err error, client *nex.Client, callID uint32, username string, key string, groups uint32, email string) {

		rmcResponseStream := nex.NewStream()

		users := database.Collection("users")
		configCollection := database.Collection("config")
		var user models.User

		// Create a new user if not currently registered.
		if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
			log.Printf("%s has never connected before - create DB entry\n", username)
			_, err := users.InsertOne(nil, bson.D{
				{Key: "username", Value: username},
				{Key: "pid", Value: config.LastPID + 1},
				// TODO: look into if the key that is passed here is per-profile, could use it as form of auth if so
			})

			if err != nil {
				log.Printf("Could not create Nintendo user %s: %s\n", username, err)
			}

			_, err = configCollection.UpdateOne(
				nil,
				bson.M{},
				bson.D{
					{"$set", bson.D{{"last_pid", config.LastPID + 1}}},
				},
			)
			if err != nil {
				log.Println("Could not update config in database: ", err)
			}

			config.LastPID++

			if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {

				if err != nil {
					log.Printf("Could not find newly created Nintendo user: %s\n", err)
				}
			}
		}
		log.Printf("%s requesting Nintendo log in from Wii Friend Code %s, has PID %v\n", username, client.WiiFC, user.PID)

		client.Username = username

		// since the Wii doesn't try hitting RegisterEx after logging in, we have to set station URLs here
		// TODO: do this better / do this proper (there's gotta be a better way), find out how to set int_station_url
		randomRVCID := rand.Intn(250000-500) + 500
		var stationURL string = "prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(randomRVCID)

		// update station URL
		result, err := users.UpdateOne(
			nil,
			bson.M{"username": client.Username},
			bson.D{
				{"$set", bson.D{{"station_url", stationURL}}},
				{"$set", bson.D{{"int_station_url", ""}}},
			},
		)

		if err != nil {
			log.Printf("Could not update station URLs for Nintendo user %s: %s\n", username, err)
		}

		log.Printf("Updated %v station URL for %s \n", result.ModifiedCount, client.Username)

		rmcResponseStream.Grow(19)
		rmcResponseStream.WriteU32LENext([]uint32{user.PID})
		rmcResponseStream.WriteBufferString("FAKE-HMAC") // not 100% sure what this is supposed to be legitimately but the game doesn't complain if its not there

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.AccountManagementProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.AccountManagementMethodNintendoCreateAccount, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	unknownProtocolServer.UnknownMethod(func(err error, client *nex.Client, callID uint32, pid uint32) {

		log.Printf("Game made unknown request to unknown protocol for %v\n", pid)
		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(10)
		rmcResponseStream.WriteU32LENext([]uint32{0})

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse := nex.NewRMCResponse(nexproto.UnknownProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.UnknownMethod, rmcResponseBody)

		rmcResponseBytes := rmcResponse.Bytes()

		responsePacket, _ := nex.NewPacketV0(client, nil)

		responsePacket.SetVersion(0)
		responsePacket.SetSource(0x31)
		responsePacket.SetDestination(0x3F)
		responsePacket.SetType(nex.DataPacket)

		newArray := make([]byte, len(rmcResponseBytes)+1)
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes)
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	ip := os.Getenv("LISTENINGIP")
	securePort := os.Getenv("SECUREPORT")

	if ip == "" {
		log.Println("No listening IP specified for the secure server, you may experience issues connecting. Please set the LISTENINGIP environment variable.")
	}

	if securePort == "" {
		log.Fatalln("No secure server port specified, please set the SECUREPORT environment variable. The default secure server port for various platforms can be found at https://github.com/ihatecompvir/GoCentral/wiki/Game-specific-Network-Details")
	}

	nexServer.Listen(ip + ":" + securePort)
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
