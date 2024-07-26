package servers

import (
	"crypto/hmac"
	"crypto/md5"
	"log"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func RequestTicket(err error, client *nex.Client, callID uint32, userPID uint32, serverPID uint32) {

	if userPID != client.PlayerID() {
		log.Printf("Requested ticket for PID %v does not match server-assigned PID %v\n", userPID, client.PlayerID())
		SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.InvalidPID) // invalid PID error
		return
	}

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

	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	AuthServer.Send(responsePacket)
}
