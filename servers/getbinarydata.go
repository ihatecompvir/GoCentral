package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"strings"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func SanitizePath(path string) string {
	// List of invalid characters except the path separators and drive letter colon
	invalidChars := []string{"*", "?", "\"", "<", ">", "|", "\r", "\n", "\x0a", "\x0d", "\x00"}

	for _, char := range invalidChars {
		path = strings.ReplaceAll(path, char, "_")
	}

	// Block path traversal sequences
	path = strings.ReplaceAll(path, "..", "__")

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

	// try to get base path from environment variables
	basePath := os.Getenv("BASEBINARYDATAPATH")

	if basePath == "" {
		basePath = "binary_data"
	}

	var filePath string

	var platformExtension string

	// we don't want to try to send a png_ps3 to xbox and etc.
	// so we need to check the platform and set the extension accordingly
	switch client.Platform() {
	case 0:
		platformExtension = "png_xbox"
	case 1, 3:
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = setlistsCollection.FindOne(ctx, bson.M{"guid": setlistGUID}).Decode(&setlist)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Println("Setlist not found with GUID ", setlistGUID)
			} else {
				log.Println("Error finding setlist with GUID ", setlistGUID, ": ", err)
			}
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

		filePath = filepath.Join(basePath, "setlist_art", setlistGUID, fmt.Sprintf("%d.%s", revision, platformExtension))
		filePath = SanitizePath(filePath)

		log.Printf("Serving setlist art at path %v", filePath)

	case "battle_art":
		revisionFloat, ok := metadataMap["revision"].(float64)

		if !ok {
			log.Println("Error parsing revision from requested metadata")
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		revision := int64(revisionFloat)

		// battle_art can optionally have battle_id; try to get it, but don't fail if it can't be found
		battleID, _ := metadataMap["battle_id"].(float64)

		// make sure the setlist with the specified battle ID actually exists
		setlistsCollection := database.GocentralDatabase.Collection("setlists")

		var setlist models.Setlist
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = setlistsCollection.FindOne(ctx, bson.M{"setlist_id": battleID}).Decode(&setlist)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Println("Battle not found with battle ID ", battleID)
			} else {
				log.Println("Error finding battle with battle ID ", battleID, ": ", err)
			}
			SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
			return
		}

		filePath = filepath.Join(basePath, "battle_art", fmt.Sprintf("%d", int64(battleID)), fmt.Sprintf("%d.%s", revision, platformExtension))
		filePath = SanitizePath(filePath)

		log.Printf("Serving battle art at path %v", filePath)

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

		filePath = filepath.Join(basePath, "band_logo", fmt.Sprintf("%d", bandID), fmt.Sprintf("%d.%s", revision, platformExtension))
		filePath = SanitizePath(filePath)

		log.Printf("Serving band logo at path %v", filePath)

	default:
		log.Println("Unsupported type ", dataType, " in requested metadata")
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
		return
	}

	// Check if the path is still valid and under basePath
	cleanBasePath := filepath.Clean(basePath)
	cleanFilePath := filepath.Clean(filePath)

	if !strings.HasPrefix(cleanFilePath, cleanBasePath) {
		log.Println("Invalid path: ", cleanFilePath)
		SendErrorCode(SecureServer, client, nexproto.RBBinaryDataProtocolID, callID, quazal.UnknownError)
		return
	}

	// Read the file from disk
	data, err := ioutil.ReadFile(cleanFilePath)
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
