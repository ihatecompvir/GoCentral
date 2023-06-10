package servers

import (
	"crypto/rc4"
	"log"
	"os"
	"rb3server/database"
	"rb3server/models"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

var AuthServer *nex.Server
var SecureServer *nex.Server

var Config models.Config

func OnConnection(packet *nex.PacketV0) {
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
	requestDataStream := nex.NewStreamIn(decryptedRequestData, AuthServer)

	// extract the PID
	userPid := requestDataStream.ReadU32LENext(1)[0]

	// Get username for client from PID. This avoids having to grab it from the ticket
	// On Wii, the ticket does not contain the username so this is a platform-agnostic solution
	var user models.User
	users := database.RockcentralDatabase.Collection("users")
	users.FindOne(nil, bson.M{"pid": userPid}).Decode(&user)
	packet.Sender().Username = user.Username

	_ = requestDataStream.ReadU32LENext(1)[0]
	responseCheck := requestDataStream.ReadU32LENext(1)[0]

	responseValueStream := nex.NewStreamIn(make([]byte, 20), AuthServer)
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
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)
}

func SendErrorCode(server *nex.Server, client *nex.Client, protocol uint8, callID uint32, code uint32) {
	rmcResponse := nex.NewRMCResponse(protocol, callID)
	rmcResponse.SetError(code)

	rmcResponseBytes := rmcResponse.Bytes()

	responsePacket, _ := nex.NewPacketV0(client, nil)

	responsePacket.SetVersion(0)
	responsePacket.SetSource(0x31)
	responsePacket.SetDestination(0x3F)
	responsePacket.SetType(nex.DataPacket)

	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)
}

func StartAuthServer() {
	AuthServer = nex.NewServer()

	AuthServer.SetPrudpVersion(0)
	AuthServer.SetSignatureVersion(1)
	AuthServer.SetKerberosKeySize(16)
	AuthServer.SetChecksumVersion(1)
	AuthServer.UsePacketCompression(false)
	AuthServer.SetFlagsVersion(0)
	AuthServer.SetAccessKey(os.Getenv("ACCESSKEY"))
	AuthServer.SetFragmentSize(750)

	authenticationProtocol := nexproto.NewAuthenticationProtocol(AuthServer)

	authenticationProtocol.Login(Login)
	authenticationProtocol.RequestTicket(RequestTicket)

	ip := os.Getenv("LISTENINGIP")
	authPort := os.Getenv("AUTHPORT")

	if ip == "" {
		log.Println("No listening IP specified for the auth server, you may experience issues connecting. Please set the LISTENINGIP environment variable.")
	}

	if authPort == "" {
		log.Fatalln("No auth server port specified, please set the AUTHPORT environment variable. The default auth server port for various platforms can be found at https://github.com/ihatecompvir/GoCentral/wiki/Game-specific-Network-Details")
	}

	AuthServer.Listen(ip + ":" + authPort)
}

func StartSecureServer() {
	SecureServer = nex.NewServer()

	SecureServer.SetPrudpVersion(0)
	SecureServer.SetSignatureVersion(1)
	SecureServer.SetKerberosKeySize(16)
	SecureServer.SetChecksumVersion(1)
	SecureServer.UsePacketCompression(false)
	SecureServer.SetFlagsVersion(0)
	SecureServer.SetAccessKey(os.Getenv("ACCESSKEY"))
	SecureServer.SetFragmentSize(750)

	secureProtocol := nexproto.NewSecureProtocol(SecureServer)
	jsonProtocol := nexproto.NewJsonProtocol(SecureServer)
	matchmakingProtocol := nexproto.NewMatchmakingProtocol(SecureServer)
	customMatchmakingProtocol := nexproto.NewCustomMatchmakingProtocol(SecureServer)
	natTraversalProtocol := nexproto.NewNATTraversalProtocol(SecureServer)
	accountManagementProtocol := nexproto.NewAccountManagementProtocol(SecureServer)
	messagingProtocolProtocol := nexproto.NewMessagingProtocol(SecureServer)

	SecureServer.On("Connect", OnConnection)

	secureProtocol.RegisterEx(RegisterEx)
	secureProtocol.RequestURLs(RequestURLs)

	jsonProtocol.JSONRequest(JSONRequest)
	jsonProtocol.JSONRequest2(JSONRequest2)

	customMatchmakingProtocol.CustomFind(CustomFind)

	matchmakingProtocol.LaunchSession(LaunchSession)
	matchmakingProtocol.Participate(Participate)
	matchmakingProtocol.Unparticipate(Unparticipate)
	matchmakingProtocol.RegisterGathering(RegisterGathering)
	matchmakingProtocol.TerminateGathering(TerminateGathering)
	matchmakingProtocol.UpdateGathering(UpdateGathering)
	matchmakingProtocol.SetState(SetState)

	natTraversalProtocol.RequestProbeInitiation(RequestProbeInitiation)

	accountManagementProtocol.NintendoCreateAccount(NintendoCreateAccount)
	accountManagementProtocol.SetStatus(SetStatus)

	messagingProtocolProtocol.GetMessageHeaders(GetMessageHeaders)

	ip := os.Getenv("LISTENINGIP")
	securePort := os.Getenv("SECUREPORT")

	if ip == "" {
		log.Println("No listening IP specified for the secure server, you may experience issues connecting. Please set the LISTENINGIP environment variable.")
	}

	if securePort == "" {
		log.Fatalln("No secure server port specified, please set the SECUREPORT environment variable. The default secure server port for various platforms can be found at https://github.com/ihatecompvir/GoCentral/wiki/Game-specific-Network-Details")
	}

	SecureServer.Listen(ip + ":" + securePort)
}
