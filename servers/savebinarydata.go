package servers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"rb3server/quazal"
	"strings"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
)

func SaveBinaryData(err error, client *nex.Client, callID uint32, metadata string, data []byte) {

	res, _ := ValidateClientPID(SecureServer, client, callID, nexproto.RBBinaryDataProtocolID)

	if !res {
		return
	}

	var metadataMap map[string]interface{}
	err = json.Unmarshal([]byte(metadata), &metadataMap)

	if err != nil {
		log.Println("Error parsing metadata: ", err)
		// print metadata
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
		return
	}

	// get the type
	dataType, ok := metadataMap["type"].(string)

	if !ok {
		log.Println("Error parsing type from requested metadata")
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
		return
	}

	// try to get base path from environment variables
	basePath := os.Getenv("BASEBINARYDATAPATH")

	if basePath == "" {
		basePath = "binary_data"
	}

	var filePath string

	// switch on the type
	switch dataType {
	case "setlist_art":
		// get the setlist guid
		setlistGUID, ok := metadataMap["setlist_guid"].(string)

		if !ok {
			log.Println("Error parsing setlist_guid from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
			return
		}

		// get the revision
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
			return
		}

		// convert float64 to int64
		revision := int64(revisionFloat)

		filePath = filepath.Join(basePath, "setlist_art", setlistGUID, fmt.Sprintf("%d.dxt", revision))

	case "battle_art":
		// get the revision
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
			return
		}

		// convert float64 to int64
		revision := int64(revisionFloat)

		// battle_art can optionally have battle_id; try to get it, but dont fail if it cant be found
		battleID, _ := metadataMap["battle_id"].(float64)

		filePath = filepath.Join(basePath, "battle_art", fmt.Sprintf("%d", int64(battleID)), fmt.Sprintf("%d.dxt", revision))

	case "band_logo":
		// get the band id
		bandIDFloat, ok := metadataMap["band_id"].(float64)

		if !ok {
			log.Println("Error parsing band_id from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
			return
		}

		// convert float64 to int64
		bandID := int64(bandIDFloat)

		// get the revision
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
			return
		}

		// convert float64 to int64
		revision := int64(revisionFloat)

		filePath = filepath.Join(basePath, "band_logo", fmt.Sprintf("%d", bandID), fmt.Sprintf("%d.dxt", revision))

	default:
		log.Printf("Unsupported type %s in requested metadata", dataType)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
		return
	}

	// Sanitize and validate the path
	filePath = SanitizePath(filePath)

	// Check if the path is still valid and under basePath
	cleanBasePath := filepath.Clean(basePath)
	cleanFilePath := filepath.Clean(filePath)

	if !strings.HasPrefix(cleanFilePath, cleanBasePath) {
		log.Println("Invalid path: ", cleanFilePath)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
		return
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(cleanFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Println("Error creating directory: ", err)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
		return
	}

	if err := os.WriteFile(cleanFilePath, data, 0644); err != nil {
		log.Println("Error writing file: ", err)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.OperationError)
		return
	}

	log.Printf("Successfully saved binary data to %s", cleanFilePath)

	rmcResponseStream := nex.NewStream()

	rmcResponseStream.WriteBufferString("(test 0)")
	rmcResponseStream.WriteUInt8(0)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.RBBinaryDataProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SaveBinaryData, rmcResponseBody)

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
