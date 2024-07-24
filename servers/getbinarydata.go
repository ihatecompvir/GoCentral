package servers

import (
	"encoding/json"
	"log"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func GetBinaryData(err error, client *nex.Client, callID uint32, metadata string) {
	// metadata looks like this
	// "{\"type\": \"setlist_art\", \"setlist_guid\": \"%s\", \"revision\": %d}"
	// "{\"type\": \"battle_art\", \"revision\": 1 }"
	// "{\"type\": \"band_logo\", \"band_id\": %d, \"revision\": %d }"
	// so we need to parse this

	// parse the json
	var metadataMap map[string]interface{}
	err = json.Unmarshal([]byte(metadata), &metadataMap)

	if err != nil {
		log.Println("Error parsing metadata: ", err)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
		return
	}

	// get the type
	dataType, ok := metadataMap["type"].(string)

	if !ok {
		log.Println("Error parsing type from requested metadata")
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
		return
	}

	// switch on the type
	switch dataType {
	case "setlist_art":
		// get the setlist guid
		setlistGUID, ok := metadataMap["setlist_guid"].(string)

		if !ok {
			log.Println("Error parsing setlist_guid from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
			return
		}

		// get the revision
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
			return
		}

		// convert float64 to int64
		revision := int64(revisionFloat)

		log.Println("Getting setlist art for setlist %s with revision %d", setlistGUID, revision)

	case "battle_art":
		// get the revision
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
			return
		}

		// convert float64 to int64
		revision := int64(revisionFloat)

		// battle_art can optionally have battle_id; try to get it, but dont fail if it cant be found
		battleID, _ := metadataMap["battle_id"].(float64)

		log.Println("Getting battle art with revision %d with battle ID %d", revision, int64(battleID))

	case "band_logo":
		// get the band id
		bandIDFloat, ok := metadataMap["band_id"].(float64)

		if !ok {
			log.Println("Error parsing band_id from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
			return
		}

		// convert float64 to int64
		bandID := int64(bandIDFloat)

		// get the revision
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
			return
		}

		// convert float64 to int64
		revision := int64(revisionFloat)

		log.Println("Getting band logo for band %d with revision %d", bandID, revision)

	default:
		log.Println("Unsupported type %s in requested metadata", dataType)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
		return
	}

	rmcResponseStream := nex.NewStream()

	if err != nil {
		log.Println("Error reading test.png_ps3: ", err)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
		return
	}

	rmcResponseStream.Grow(10)

	// TODO: implement loading binary data from disk
	rmcResponseStream.WriteBuffer([]byte{0x00})
	rmcResponseStream.WriteBufferString("{}") // the game doesn't really care what we send here so just send empty json

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.RBBinaryDataProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.GetBinaryData, rmcResponseBody)

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
