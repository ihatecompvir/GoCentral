package servers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func SaveBinaryData(err error, client *nex.Client, callID uint32, metadata string, data []byte) {

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

		os.WriteFile(fmt.Sprintf("./binary_data/setlist_art/%s/%d.dxt", setlistGUID, revision), data, 0644)

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

		os.WriteFile(fmt.Sprintf("./binary_data/battle_art/%d/%d.dxt", int64(battleID), revision), data, 0644)

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

		os.WriteFile(fmt.Sprintf("./binary_data/band_logo/%d/%d.dxt", bandID, revision), data, 0644)

	default:
		log.Printf("Unsupported type %s in requested metadata", dataType)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, 0x00010001)
		return
	}

	rmcResponseStream := nex.NewStream()

	rmcResponseStream.Grow(10)

	rmcResponseStream.WriteBufferString("{}") // the game doesn't really care what we send here so just send empty json
	rmcResponseStream.WriteUInt8(0)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.RBBinaryDataProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SaveBinaryData, rmcResponseBody)

	responsePacket, _ := nex.NewPacketV0(client, nil)

	responsePacket.SetVersion(0)
	responsePacket.SetSource(0x31)
	responsePacket.SetDestination(0x3F)
	responsePacket.SetType(nex.DataPacket)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	SecureServer.Send(responsePacket)

}
