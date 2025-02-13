package servers

import (
	"log"
	"rb3server/quazal"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func RequestProbeInitiation(err error, client *nex.Client, callID uint32, stationURLs []string) {

	// check that client is not nil
	if client == nil {
		log.Println("Client is nil, cannot perform NAT probe")
		return
	}

	res, _ := ValidateNonMasterClientPID(SecureServer, client, callID, nexproto.NATTraversalProtocolID)

	if !res {
		return
	}

	log.Printf("Client wants to perform NAT traversal probes to %v servers...\n", len(stationURLs))

	// make sure we aren't trying to probe more than 8 station URLs
	// RB3 is limited to 8 player lobbies, but I believe the game can probe both the internal and external station URLs of each player
	// so 8 should be a sufficient cap
	if len(stationURLs) > 8 {
		log.Println("Client is attempting to probe more than 8 servers, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.InvalidArgument)
		return
	}

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

		// sanity check on station URL length
		if len(target) > 256 {
			log.Println("Station URL is too long, skipping probe")
			continue
		}

		targetUrl := nex.NewStationURL(target)

		if targetUrl == nil {
			log.Println("Could not parse station URL, skipping probe")
			continue
		}

		log.Println("Sending NAT probe to " + target)
		targetClient := SecureServer.FindClientFromIPAddress(targetUrl.Address() + ":" + targetUrl.Port())
		if targetClient != nil {
			var messagePacket nex.PacketInterface

			messagePacket, _ = nex.NewPacketV0(targetClient, nil)

			log.Println("Found active client " + targetClient.ExternalStationURL() + " with RVCID " + targetUrl.RVCID() + " and username " + targetClient.Username + " and IP address " + targetClient.Address().IP.String())
			messagePacket.SetVersion(0)

			messagePacket.SetSource(0x31)
			messagePacket.SetDestination(0x3F)
			messagePacket.SetType(nex.DataPacket)

			messagePacket.SetPayload(rmcMessageBytes)
			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)

			SecureServer.Send(messagePacket)
		} else {
			log.Printf("Could not find active client with IP %v, skipping probe\n", targetUrl.Address()+":"+targetUrl.Port())
			continue
		}
	}

}
