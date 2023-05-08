package servers

import (
	"log"
	"strconv"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func RequestProbeInitiation(err error, client *nex.Client, callID uint32, stationURLs []string) {

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to initiate a NAT probe without a valid server-assigned PID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, 0x00010001)
		return
	}

	log.Printf("Client wants to perform NAT traversal probes to %v servers...\n", len(stationURLs))

	rmcResponseStream := nex.NewStream()

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.NATTraversalProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.RequestProbeInitiation, rmcResponseBody)

	responsePacket, _ := nex.NewPacketV0(client, nil)

	responsePacket.SetVersion(0)
	responsePacket.SetSource(0x31)
	responsePacket.SetDestination(0x3F)
	responsePacket.SetType(nex.DataPacket)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)

	rmcMessage := nex.RMCRequest{}
	rmcMessage.SetProtocolID(nexproto.NATTraversalProtocolID)
	rmcMessage.SetCallID(callID)
	rmcMessage.SetMethodID(nexproto.InitiateProbe)
	rmcRequestStream := nex.NewStreamOut(SecureServer)
	rmcRequestStream.WriteBufferString(client.ExternalStationURL())
	rmcRequestBody := rmcRequestStream.Bytes()
	rmcMessage.SetParameters(rmcRequestBody)
	rmcMessageBytes := rmcMessage.Bytes()

	// loop through every station URL in the probe request and send InitiateProbe to them
	// This should make all targets respond to NAT probes from the joining client
	for _, target := range stationURLs {
		targetUrl := nex.NewStationURL(target)
		log.Println("Sending NAT probe to " + target)
		targetRvcID, _ := strconv.Atoi(targetUrl.RVCID())
		targetClient := SecureServer.FindClientFromConnectionID(uint32(targetRvcID))
		if targetClient != nil {
			var messagePacket nex.PacketInterface

			messagePacket, _ = nex.NewPacketV0(targetClient, nil)
			messagePacket.SetVersion(0)

			messagePacket.SetSource(0x31)
			messagePacket.SetDestination(0x3F)
			messagePacket.SetType(nex.DataPacket)

			messagePacket.SetPayload(rmcMessageBytes)
			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)

			SecureServer.Send(messagePacket)
		} else {
			log.Printf("Could not find active client with RVCID %v\n", targetRvcID)
			SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, 0x00010001)
			return
		}
	}

}
