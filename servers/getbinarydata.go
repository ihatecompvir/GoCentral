package servers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"strings"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

func SanitizePath(path string) string {
	// check for any invalid characters in the path
	// if any are found, replace them with an underscore
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", "\\r", "\\n", "\x0a", "\x0d", ".", "\x00"}

	for _, char := range invalidChars {
		path = strings.ReplaceAll(path, char, "_")
	}

	return path
}

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
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
		return
	}

	// get the type
	dataType, ok := metadataMap["type"].(string)

	if !ok {
		log.Println("Error parsing type from requested metadata")
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
		return
	}

	var filePath string

	var platformExtension string

	// we don't want to try to send a png_ps3 to xbox and etc.
	// so we need to check the platform and set the extension accordingly
	switch client.Platform() {
	case 0:
		platformExtension = "png_xbox"
	case 1:
	case 3:
		platformExtension = "png_ps3"
	case 2:
		platformExtension = "png_wii"
	default:
		log.Printf("Unsupported platform %d in requested metadata", client.Platform())
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
		return
	}

	switch dataType {
	case "setlist_art":
		setlistGUID, ok := metadataMap["setlist_guid"].(string)

		if !ok {
			log.Println("Error parsing setlist_guid from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		// make sure the setlist with the specified GUID actually exists
		setlistsCollection := database.GocentralDatabase.Collection("setlists")

		var setlist models.Setlist
		err = setlistsCollection.FindOne(nil, bson.M{"guid": setlistGUID}).Decode(&setlist)

		if err != nil {
			log.Println("Error finding setlist with GUID ", setlistGUID)
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		revision := int64(revisionFloat)

		filePath = fmt.Sprintf("binary_data/setlist_art/%s/%d."+platformExtension, setlistGUID, revision)

		filePath = filepath.Clean(SanitizePath(filePath))

		if !strings.HasPrefix(filePath, "binary_data") {
			log.Println("Invalid path: ", filePath)
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

	case "battle_art":
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		revision := int64(revisionFloat)

		// battle_art can optionally have battle_id; try to get it, but dont fail if it cant be found
		battleID, _ := metadataMap["battle_id"].(float64)

		// make sure the setlist with the specified battle ID actually exists
		setlistsCollection := database.GocentralDatabase.Collection("setlists")

		var setlist models.Setlist
		err = setlistsCollection.FindOne(nil, bson.M{"setlist_id": battleID}).Decode(&setlist)

		if err != nil {
			log.Println("Error finding battle with battle ID ", battleID)
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		filePath = fmt.Sprintf("binary_data/battle_art/%d/%d."+platformExtension, int64(battleID), revision)

		filePath = filepath.Clean(SanitizePath(filePath))

		if !strings.HasPrefix(filePath, "binary_data") {
			log.Println("Invalid path: ", filePath)
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

	case "band_logo":
		bandIDFloat, ok := metadataMap["band_id"].(float64)

		if !ok {
			log.Println("Error parsing band_id from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		bandID := int64(bandIDFloat)

		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		revision := int64(revisionFloat)

		filePath = fmt.Sprintf("binary_data/band_logo/%d/%d."+platformExtension, bandID, revision)

		filePath = filepath.Clean(SanitizePath(filePath))

		if !strings.HasPrefix(filePath, "binary_data") {
			log.Println("Invalid path: ", filePath)
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

	default:
		log.Println("Unsupported type %s in requested metadata", dataType)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
		return
	}

	// Read the file from disk
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println("Error reading file from disk: ", err)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
		return
	}

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.WriteBuffer(data)
	rmcResponseStream.WriteBufferString("{}")

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
