package servers

import (
	"log"
	"net"
	"rb3server/quazal"
	"strconv"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func RequestProbeInitiation(err error, client *nex.Client, callID uint32, stationURLs []string) {

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to initiate a NAT probe without a valid server-assigned PID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.NotAuthenticated)
		return
	}

	log.Printf("Client wants to perform NAT traversal probes to %v servers...\n", len(stationURLs))

	// make sure we aren't trying to probe more than 8 station URLs
	// RB3 is limited to 4 player lobbies, but I believe the game can probe both the internal and external station URLs of each player
	// so 8 should be a sufficient cap
	if len(stationURLs) > 4 {
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
			log.Println("Station URL is too long, rejecting call")
			SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.InvalidArgument)
			return
		}

		targetUrl := nex.NewStationURL(target)

		// ensure that the station URL has a port, scheme, and URL
		if targetUrl.Port() == "" || targetUrl.Scheme() == "" || targetUrl.Address() == "" {
			log.Println("Station URL is missing a port, scheme, or URL, rejecting call")
			SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.InvalidArgument)
			return
		}

		targetPort, err := strconv.Atoi(targetUrl.Port())
		if err != nil {
			log.Println("Error converting port to int: " + err.Error())
			SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.InvalidArgument)
			return
		}

		targetIP := net.ParseIP(targetUrl.Address())
		if targetIP == nil {
			log.Println("Invalid IP address: " + targetUrl.Address())
			SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.InvalidArgument)
			return
		}

		addr := net.UDPAddr{IP: targetIP, Port: targetPort}

		targetClient := nex.NewClient(&addr, SecureServer)
		if targetClient == nil {
			log.Println("Error creating client for target: " + targetUrl.Address())
			SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.UnknownError)
			return
		}

		var messagePacket nex.PacketInterface

		messagePacket, err = nex.NewPacketV0(targetClient, nil)
		if err != nil {
			log.Println("Error creating packet: " + err.Error())
			SendErrorCode(SecureServer, client, nexproto.NATTraversalProtocolID, callID, quazal.UnknownError)
			return
		}

		log.Println("Sending NAT probe to " + targetUrl.Address() + ":" + targetUrl.Port())
		messagePacket.SetVersion(0)

		messagePacket.SetSource(0x31)
		messagePacket.SetDestination(0x3F)
		messagePacket.SetType(nex.DataPacket)

		messagePacket.SetPayload(rmcMessageBytes)
		messagePacket.AddFlag(nex.FlagNeedsAck)
		messagePacket.AddFlag(nex.FlagReliable)

		SecureServer.Send(messagePacket)
	}

}
