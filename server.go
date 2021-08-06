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
	"strings"
	"syscall"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	uri := os.Getenv("MONGOCONNECTIONSTRING")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {
		panic(err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected and pinged.")

	gocentralDatabase := client.Database("gocentral")

	go mainAuth(gocentralDatabase)
	go mainSecure(gocentralDatabase)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
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

	authenticationServer := nexproto.NewAuthenticationProtocol(nexServer)

	authenticationServer.Login(func(err error, client *nex.Client, callID uint32, username string) {
		serverPID := 2 // Quazal Rendez-Vous

		users := database.Collection("users")

		var user models.User

		if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
			fmt.Printf("%s has never connected before - create DB entry\n", username)
			_, err := users.InsertOne(nil, bson.D{
				{Key: "username", Value: username},
				{Key: "pid", Value: rand.Intn(250000-500) + 500},
			})

			if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {

				if err != nil {
					log.Fatal(err)
				}
			}
		}

		client.Username = username

		fmt.Printf("%s requesting log in, has PID %v\n", username, user.PID)
		encryptedTicket, kerberosKey := generateKerberosTicket(user.PID, uint32(serverPID), 16)
		mac := hmac.New(md5.New, kerberosKey)
		mac.Write(encryptedTicket)
		calculatedHmac := mac.Sum(nil)

		// Build the response body
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
		fmt.Printf("PID %v requesting ticket...\n", userPID)

		encryptedTicket, kerberosKey := generateKerberosTicket(userPID, uint32(serverPID), 16)
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

	nexServer.Listen("0.0.0.0:" + os.Getenv("AUTHPORT"))

}

func mainSecure(database *mongo.Database) {
	nexServer := nex.NewServer()
	nexServer.SetPrudpVersion(0)
	nexServer.SetSignatureVersion(1)
	nexServer.SetKerberosKeySize(16)
	nexServer.SetChecksumVersion(1)
	nexServer.UsePacketCompression(true)
	nexServer.SetFlagsVersion(0)
	nexServer.SetAccessKey(os.Getenv("ACCESSKEY"))

	secureServer := nexproto.NewSecureProtocol(nexServer)
	jsonServer := nexproto.NewJsonProtocol(nexServer)
	matchmakingServer := nexproto.NewMatchmakingProtocol(nexServer)
	natTraversalServer := nexproto.NewNATTraversalProtocol(nexServer)

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

		_ = requestDataStream.ReadU32LENext(1)[0] // User PID
		_ = requestDataStream.ReadU32LENext(1)[0] //CID of secure server station url
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

		// get the username from the submitted NPTicket so we can lookup PID and write station URL
		username := string(ticketData[92:108])
		username = strings.TrimRight(username, "\x00")

		var user models.User

		if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
			log.Fatal(err)
		}

		// Build the response body
		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(200)

		rmcResponseStream.WriteU16LENext([]uint16{0x01})
		rmcResponseStream.WriteU16LENext([]uint16{0x01})
		rmcResponseStream.WriteU32LENext([]uint32{user.PID}) // pid

		randomRVCID := rand.Intn(250000-500) + 500
		var stationURL string = "prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(randomRVCID)

		re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)

		submatchall := re.FindAllString(stationUrls[0], -1)
		var internalStationURL string = "prudp:/address=" + submatchall[0] + ";port=" + fmt.Sprint(client.Address().Port) + ";PID=" + fmt.Sprint(user.PID) + ";sid=15;type=3;RVCID=" + fmt.Sprint(randomRVCID)

		// update station URL
		result, err := users.UpdateOne(
			nil,
			bson.M{"username": username},
			bson.D{
				{"$set", bson.D{{"station_url", stationURL}}},
				{"$set", bson.D{{"int_station_url", internalStationURL}}},
			},
		)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Updated %v station URL for %s \n", result.ModifiedCount, username)
		client.Username = username

		// The game doesn't appear to do anything with this at first glance, but return something proper anyway
		rmcResponseStream.WriteBufferString("prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";sid=15;type=3;RVCID=" + fmt.Sprint(randomRVCID))

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

	secureServer.RequestURLs(func(err error, client *nex.Client, callID uint32, stationCID uint32, stationPID uint32) {
		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		fmt.Println(stationCID)
		fmt.Printf("Requesting station URL for %v\n", stationPID)

		users := database.Collection("users")

		var user models.User

		if err = users.FindOne(nil, bson.M{"pid": stationPID}).Decode(&user); err != nil {
			log.Fatal(err)
		}

		rmcResponseStream.WriteUInt8(1)
		rmcResponseStream.WriteU32LENext([]uint32{2})
		rmcResponseStream.WriteBufferString(user.StationURL)
		rmcResponseStream.WriteBufferString(user.IntStationURL)

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
			fmt.Printf("failed to handle request: %+v\n", err)
			res = "[]"
		} else {
			fmt.Printf("in:\n%s\n", rawJson)
			fmt.Printf("out:\n%s\n", res)
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
		fmt.Println("Registering gathering...")

		// delete old gatherings, and create a new gathering

		gatherings := database.Collection("gatherings")

		gatheringID := rand.Intn(250000-500) + 500

		deleteResult, deleteError := gatherings.DeleteMany(nil, bson.D{
			{Key: "creator", Value: client.Username},
		})

		if deleteError != nil {
			fmt.Println("Could not clear stale gatherings")
		}

		if deleteResult.DeletedCount != 0 {
			fmt.Printf("Successfully cleared %v stale gatherings for %s...\n", deleteResult.DeletedCount, client.Username)
		}

		res, err := gatherings.InsertOne(nil, bson.D{
			{Key: "gathering_id", Value: gatheringID},
			{Key: "contents", Value: gathering},
			{Key: "creator", Value: client.Username},
		})

		fmt.Println(res)

		if err != nil {
			log.Fatal("Could not create gathering")
		}

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteU32LENext([]uint32{uint32(gatheringID)})

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
		fmt.Printf("Updating gathering for %s\n", client.Username)

		gatherings := database.Collection("gatherings")

		//var harmonixGathering models.Gathering

		result, err := gatherings.UpdateOne(
			nil,
			bson.M{"gathering_id": gatheringID},
			bson.D{
				{"$set", bson.D{{"contents", gathering}}},
			},
		)

		if err != nil {
			log.Fatal("Could not update gathering")
			return
		}

		fmt.Printf("Updated %v gatherings\n", result.ModifiedCount)

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

		rmcResponseStream.WriteU32LENext([]uint32{gatheringID})
		rmcResponseStream.WriteU32LENext([]uint32{1})

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

		rmcResponseStream.WriteU32LENext([]uint32{gatheringID})
		rmcResponseStream.WriteU32LENext([]uint32{1})

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
		fmt.Printf("Launching session for %s...\n", client.Username)

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteU32LENext([]uint32{1})

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
		fmt.Printf("Terminating gathering for %s...\n", client.Username)

		gatherings := database.Collection("gatherings")

		result, err := gatherings.DeleteOne(
			nil,
			bson.M{"gathering_id": gatheringID},
		)

		if err != nil {
			log.Fatal("Could not delete gathering")
			return
		}

		fmt.Printf("Terminated %v gathering\n", result.DeletedCount)

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(50)

		rmcResponseStream.WriteU32LENext([]uint32{1})

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
		fmt.Printf("Checking for available gatherings for %s...\n", client.Username)

		gatherings := database.Collection("gatherings")
		users := database.Collection("users")

		var gathering models.Gathering
		var user models.User

		gatherings.FindOne(nil,
			bson.D{{
				Key:   "creator",
				Value: bson.D{{Key: "$ne", Value: client.Username}},
			}},
		).Decode(&gathering)

		//if err != nil {
		//	log.Fatal("Could not delete gathering")
		//}

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(19)

		if len(gathering.Contents) == 0 {
			fmt.Println("There are no active gatherings. Tell client to keep checking.")
			rmcResponseStream.WriteU32LENext([]uint32{0})
		} else {
			fmt.Println("Found a gathering - attempting join!")
			if err = users.FindOne(nil, bson.M{"username": gathering.Creator}).Decode(&user); err != nil {
				log.Fatal(err)
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

	natTraversalServer.InitiateProbe(func(err error, client *nex.Client, callID uint32, stationURL string) {
		fmt.Printf("Doing NAT traversal probe for  %s...\n", stationURL)

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

	nexServer.Listen("0.0.0.0:" + os.Getenv("SECUREPORT"))
}

func generateKerberosTicket(userPID uint32, serverPID uint32, keySize int) ([]byte, []byte) {

	sessionKey := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10}

	// Create ticket body info
	kerberosTicketInfoKey := make([]byte, 16)
	//rand.Read(kerberosTicketInfoKey) // TODO: enable random keys and make them shared with secure server

	ticketInfoEncryption := nex.NewKerberosEncryption(kerberosTicketInfoKey)
	ticketInfoStream := nex.NewStream()

	encryptedTicketInfo := ticketInfoEncryption.Encrypt(ticketInfoStream.Bytes())

	// Create ticket
	kerberosTicketKey := deriveKerberosKey(userPID)

	ticketEncryption := nex.NewKerberosEncryption(kerberosTicketKey)
	ticketStream := nex.NewStream()
	ticketStream.Grow(int64(24))

	ticketStream.WriteBytesNext(sessionKey)
	ticketStream.WriteU32LENext([]uint32{1})
	ticketStream.WriteU32LENext([]uint32{0x24})
	ticketStream.WriteBuffer(encryptedTicketInfo)
	return ticketEncryption.Encrypt(ticketStream.Bytes()), kerberosTicketKey
}

func deriveKerberosKey(userPID uint32) []byte {
	// hardcoded dummy pwd, only guest doesn't use this password
	kerberosTicketKey := []byte("PS3NPDummyPwd")

	for i := 0; i < 65000+(int(userPID)%1024); i++ {
		kerberosTicketKey = nex.MD5Hash(kerberosTicketKey)
	}

	return kerberosTicketKey
}
