package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rc4"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func main() {
	go mainAuth()
	go mainSecure()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}

func mainAuth() {
	nexServer := nex.NewServer()

	nexServer.SetPrudpVersion(0)
	nexServer.SetSignatureVersion(1)
	nexServer.SetKerberosKeySize(16)
	nexServer.SetChecksumVersion(1)
	nexServer.UsePacketCompression(false)
	nexServer.SetFlagsVersion(0)
	nexServer.SetAccessKey("bfa620c57c2d3bcdf4362a6fa6418e58")

	authenticationServer := nexproto.NewAuthenticationProtocol(nexServer)

	authenticationServer.Login(func(err error, client *nex.Client, callID uint32, username string) {
		userPID, _ := strconv.Atoi(username)
		serverPID := 2 // Quazal Rendez-Vous

		encryptedTicket, kerberosKey := generateKerberosTicket(uint32(12345), uint32(serverPID), 16)
		mac := hmac.New(md5.New, kerberosKey)
		mac.Write(encryptedTicket)
		calculatedHmac := mac.Sum(nil)

		fmt.Println(userPID)

		// Build the response body
		stationURL := fmt.Sprintf("prudps:/address=%s;port=16016;CID=1;PID=2;sid=1;stream=3;type=2", os.Getenv("ADDRESS"))

		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(int64(23))

		rmcResponseStream.WriteU32LENext([]uint32{0x10001}) // success
		rmcResponseStream.WriteU32LENext([]uint32{12345})
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
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes[0:len(rmcResponseBytes)])
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)

	})

	authenticationServer.RequestTicket(func(err error, client *nex.Client, callID uint32, userPID uint32, serverPID uint32) {
		encryptedTicket, kerberosKey := generateKerberosTicket(uint32(12345), uint32(serverPID), 16)
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
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes[0:len(rmcResponseBytes)])
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	nexServer.Listen("0.0.0.0:16015")

}

func mainSecure() {
	nexServer := nex.NewServer()
	nexServer.SetPrudpVersion(0)
	nexServer.SetSignatureVersion(1)
	nexServer.SetKerberosKeySize(16)
	nexServer.SetChecksumVersion(1)
	nexServer.UsePacketCompression(true)
	nexServer.SetFlagsVersion(0)
	nexServer.SetAccessKey("bfa620c57c2d3bcdf4362a6fa6418e58")

	secureServer := nexproto.NewSecureProtocol(nexServer)
	jsonServer := nexproto.NewJsonProtocol(nexServer)

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
		fmt.Printf("Request data: %v\n", requestData)

		// TODO: use random key from auth server
		sessionKey := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10}

		requestDataEncryption, error := rc4.NewCipher(sessionKey)
		fmt.Println(error)
		decryptedRequestData := make([]byte, 0x1C)
		requestDataEncryption.XORKeyStream(decryptedRequestData, requestData)
		requestDataStream := nex.NewStreamIn(decryptedRequestData, nexServer)

		pid := requestDataStream.ReadU32LENext(1)[0] // User PID
		fmt.Printf("User PID: %v\n", pid)
		_ = requestDataStream.ReadU32LENext(1)[0] //CID of secure server station url
		responseCheck := requestDataStream.ReadU32LENext(1)[0]
		fmt.Printf("Response check: %v\n", responseCheck)

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

	secureServer.RegisterEx(func(err error, client *nex.Client, callID uint32, stationUrls []*nex.StationURL, className string, ticketData []byte) {

		// Build the response body
		rmcResponseStream := nex.NewStream()
		rmcResponseStream.Grow(200)

		rmcResponseStream.WriteU16LENext([]uint16{0x01})
		rmcResponseStream.WriteU16LENext([]uint16{0x01})
		rmcResponseStream.WriteU32LENext([]uint32{12345}) // pid

		// The game doesn't appear to do anything with this at first glance, but return something proper anyway
		rmcResponseStream.WriteBufferString("prudp:/address=" + client.Address().IP.String() + ";port=" + fmt.Sprint(client.Address().Port) + ";sid=15;type=3")

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
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes[0:len(rmcResponseBytes)])
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	jsonServer.JSONRequest(func(err error, client *nex.Client, callID uint32, rawJson string) {
		fmt.Println(rawJson)

		rmcResponseStream := nex.NewStream()

		dta := "[ [\"config/get\", \"ss\", [\"out_dta\", \"version\"], [ [\"{do {main_hub_panel set_motd \\\"Connected to GoCentral servers. The current date is " + time.Now().Format("01-02-2006") + ".\\\"} {main_hub_panel set_dlcmotd \\\"Hello.\\\"} }\", \"3\"]] ] ]"

		rmcResponseStream.WriteBufferString(dta)

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
		copy(newArray[1:len(rmcResponseBytes)+1], rmcResponseBytes[0:len(rmcResponseBytes)])
		responsePacket.SetPayload(newArray)

		responsePacket.AddFlag(nex.FlagNeedsAck)

		nexServer.Send(responsePacket)
	})

	nexServer.Listen("0.0.0.0:16016")
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
