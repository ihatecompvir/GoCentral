package servers

import (
	"log"

	"rb3server/protocols/jsonproto"

	"github.com/knvtva/nex-go"
	nexproto "github.com/knvtva/nex-protocols-go"
)

var jsonMgr = jsonproto.NewServicesManager()

func JSONRequest(err error, client *nex.Client, callID uint32, rawJson string) {

	if client.PlayerID() == 0 {
		log.Println("Client is attempting to call JSON method without valid server-assigned PID, rejecting call")
	}

	// the JSON server will handle the request depending on what needs to be returned
	res, err := jsonMgr.Handle(rawJson, client)
	if err != nil {
		SendErrorCode(SecureServer, client, nexproto.JsonProtocolID, callID, 0x00010001)
		return
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

	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)
}

func JSONRequest2(err error, client *nex.Client, callID uint32, rawJson string) {

	// I believe the second request method never returns a payload
	if client.PlayerID() == 0 {
		log.Println("Client is attempting to call JSON method without valid server-assigned PID, rejecting call")
		SendErrorCode(SecureServer, client, nexproto.JsonProtocolID, callID, 0x00010001)
		return
	}

	_, err = jsonMgr.Handle(rawJson, client)
	if err != nil {
		SendErrorCode(SecureServer, client, nexproto.JsonProtocolID, callID, 0x00010001)
		return
	}

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

	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)
}
