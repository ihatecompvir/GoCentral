package main

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

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
	nexServer.UsePacketCompression(false)
	nexServer.SetFlagsVersion(0)
	nexServer.SetAccessKey("bfa620c57c2d3bcdf4362a6fa6418e58")

	// Handle PRUDP CONNECT packet (not an RMC method)
	nexServer.On("Connect", func(packet *nex.PacketV0) {
		packet.Sender().SetClientConnectionSignature(packet.Sender().ClientConnectionSignature())

		stream := nex.NewStream()

		ticketData := stream.ReadBytesNext(0x28)
		requestData := stream.ReadBytesNext(0x20)

		// TODO: use random key from auth server
		ticketDataEncryption := nex.NewKerberosEncryption([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		decryptedTicketData := ticketDataEncryption.Decrypt(ticketData)
		ticketDataStream := nex.NewStreamIn(decryptedTicketData, nexServer)

		_ = ticketDataStream.ReadU64LENext(1)[0] // expiration time
		_ = ticketDataStream.ReadU32LENext(1)[0] // User PID
		sessionKey := ticketDataStream.ReadBytesNext(16)

		requestDataEncryption := nex.NewKerberosEncryption(sessionKey)
		decryptedRequestData := requestDataEncryption.Decrypt(requestData)
		requestDataStream := nex.NewStreamIn(decryptedRequestData, nexServer)

		_ = requestDataStream.ReadU32LENext(1)[0] // User PID
		_ = requestDataStream.ReadU32LENext(1)[0] //CID of secure server station url
		responseCheck := requestDataStream.ReadU32LENext(1)[0]

		responseValueStream := nex.NewStreamIn(make([]byte, 4), nexServer)
		responseValueBufferStream := nex.NewStream()
		responseValueBufferStream.Grow(8)

		responseValueStream.WriteU32LENext([]uint32{responseCheck + 1})
		responseValueBufferStream.WriteBuffer(responseValueStream.Bytes())

		packet.Sender().UpdateRC4Key(sessionKey)

		nexServer.AcknowledgePacket(packet, responseValueBufferStream.Bytes())
	})

	nexServer.Listen("0.0.0.0:16016")
}

func generateKerberosTicket(userPID uint32, serverPID uint32, keySize int) ([]byte, []byte) {
	nexPassword := "PS3NPDummyPwd" // TODO: Get this from database

	sessionKey := make([]byte, 16)
	rand.Read(sessionKey)

	// Create ticket body info
	kerberosTicketInfoKey := make([]byte, 16)
	//rand.Read(kerberosTicketInfoKey) // TODO: enable random keys and make them shared with secure server

	ticketInfoEncryption := nex.NewKerberosEncryption(kerberosTicketInfoKey)
	ticketInfoStream := nex.NewStream()

	encryptedTicketInfo := ticketInfoEncryption.Encrypt(ticketInfoStream.Bytes())

	// Create ticket
	kerberosTicketKey := []byte(nexPassword)
	for i := 0; i < 65000+(int(userPID)%1024); i++ {
		kerberosTicketKey = nex.MD5Hash(kerberosTicketKey)
	}

	ticketEncryption := nex.NewKerberosEncryption(kerberosTicketKey)
	ticketStream := nex.NewStream()
	ticketStream.Grow(int64(24))

	ticketStream.WriteBytesNext(sessionKey)
	ticketStream.WriteU32LENext([]uint32{1})
	ticketStream.WriteU32LENext([]uint32{0x24})
	ticketStream.WriteBuffer(encryptedTicketInfo)
	return ticketEncryption.Encrypt(ticketStream.Bytes()), kerberosTicketKey
}
