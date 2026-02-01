package servers

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"rb3server/database"
	"rb3server/models"
	"rb3server/quazal"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ihatecompvir/nex-go"
	nexproto "github.com/ihatecompvir/nex-protocols-go"
	"go.mongodb.org/mongo-driver/bson"
)

var machineType int = 255 // 0 = xbox, 1 = ps3, 2 = wii

// cache entry with exp
type kerberosKeyCacheEntry struct {
	key       []byte
	expiresAt time.Time
}

// kerb cache
var (
	kerberosKeyCache   = make(map[string]*kerberosKeyCacheEntry)
	kerberosKeyCacheMu sync.RWMutex
	kerberosKeyCacheTTL = 5 * time.Minute
)

func deriveKerberosKey(userPID uint32, pwd string) []byte {
	pwdToUse := pwd
	if pwd == "" {
		pwdToUse = os.Getenv("USERPASSWORD")
	}
	cacheKey := fmt.Sprintf("%d:%s", userPID, pwdToUse)

	// check if it is in cache first
	kerberosKeyCacheMu.RLock()
	if entry, ok := kerberosKeyCache[cacheKey]; ok {
		if time.Now().Before(entry.expiresAt) {
			kerberosKeyCacheMu.RUnlock()
			return entry.key
		}
	}
	kerberosKeyCacheMu.RUnlock()

	// was not cached - compute it
	kerberosTicketKey := []byte(pwdToUse)

	for i := 0; i < 65000+(int(userPID)%1024); i++ {
		kerberosTicketKey = nex.MD5Hash(kerberosTicketKey)
	}

	// put it in cache
	kerberosKeyCacheMu.Lock()
	kerberosKeyCache[cacheKey] = &kerberosKeyCacheEntry{
		key:       kerberosTicketKey,
		expiresAt: time.Now().Add(kerberosKeyCacheTTL),
	}
	kerberosKeyCacheMu.Unlock()

	return kerberosTicketKey
}

func generateGUID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	guid := hex.EncodeToString(bytes)
	return strings.ToLower(guid), nil
}

func generateKerberosTicket(userPID uint32, serverPID uint32, keySize int, pwd string) ([]byte, []byte) {

	sessionKey := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10}

	// Create ticket body info
	kerberosTicketInfoKey := make([]byte, 16)
	//rand.Read(kerberosTicketInfoKey) // TODO: enable random keys and make them shared with secure server

	ticketInfoEncryption := nex.NewKerberosEncryption(kerberosTicketInfoKey)
	ticketInfoStream := nex.NewStream()

	encryptedTicketInfo := ticketInfoEncryption.Encrypt(ticketInfoStream.Bytes())

	// Create ticket
	kerberosTicketKey := deriveKerberosKey(userPID, pwd)

	ticketEncryption := nex.NewKerberosEncryption(kerberosTicketKey)
	ticketStream := nex.NewStream()
	ticketStream.Grow(int64(24))

	ticketStream.WriteBytesNext(sessionKey)
	ticketStream.WriteU32LENext([]uint32{1})
	ticketStream.WriteU32LENext([]uint32{0x24})
	ticketStream.WriteBuffer(encryptedTicketInfo)
	return ticketEncryption.Encrypt(ticketStream.Bytes()), kerberosTicketKey
}

func Login(err error, client *nex.Client, callID uint32, username string) {
	serverPID := 2 // Quazal Rendez-Vous

	users := database.GocentralDatabase.Collection("users")

	var user models.User

	// check for Wii FC inside parentheses
	// TODO: - do this better, this feels bleh
	var rgx = regexp.MustCompile(`\(([^()]*)\)`)
	res := rgx.FindStringSubmatch(username)

	// If there is no regex found, we are a PS3 client so get the correct stuff from the DB for the user
	// PS3 usernames cannot contain parentheses so there is no chance of a PS3 client taking the wii path

	// (TODO) Add support for RPCS3 & Dolphin (Xenia can be added once they make networking on it better.)

	if client.Server().AccessKey() == "d52d1e000328fbc724fde65006b88b56" { // xbox 360
		log.Println("Xbox client connecting")
		machineType = 0
	} else if client.Server().AccessKey() == "bfa620c57c2d3bcdf4362a6fa6418e58" {
		log.Println("PS3 client connecting")
		machineType = 1
	} else if client.Server().AccessKey() == "e97dc2ce9904698f84cae429a41b328a" {
		log.Println("Wii client connecting")
		machineType = 2
	} else {
		log.Println("Unknown machine connecting --- ABORT") // Basically it doesn't fall into this category
		SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.InvalidArgument)
		return
	}

	config, err := database.GetCachedConfig(context.TODO())
	if err != nil {
		log.Printf("Could not get config: %v\n", err)
		SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.OperationError)
		return
	}

	// check if they've got any bans
	var userBans []models.BannedPlayer
	for _, bannedPlayer := range config.BannedPlayers {
		if bannedPlayer.Username == username {
			userBans = append(userBans, bannedPlayer)
		}
	}

	// if there are any bans, check if the latest one is still active
	if len(userBans) > 0 {
		// sort bans by CreatedAt timestamp, descending
		sort.Slice(userBans, func(i, j int) bool {
			return userBans[i].CreatedAt.After(userBans[j].CreatedAt)
		})

		latestBan := userBans[0]

		// zero time means permanent ban
		// if the expires time is in the past, this is an old ban and we can ignore it
		if latestBan.ExpiresAt.IsZero() || time.Now().Before(latestBan.ExpiresAt) {
			log.Printf("Banned user %s attempted to log in. Denying connection based on ban created at %s.", username, latestBan.CreatedAt)
			SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.AccountDisabled)
			return
		}
	}

	switch machineType {
	case 0, 1:
		if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
			log.Printf("%s has never connected before - create DB entry\n", username)

			guid, err := generateGUID()

			// get nex pid atomically to avoid racing with multiple server instances
			newPID, err := database.GetNextPID(context.TODO())
			if err != nil {
				log.Printf("Could not get next PID: %s\n", err)
				SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.OperationError)
				return
			}

			_, err = users.InsertOne(nil, bson.D{
				{Key: "username", Value: username},
				{Key: "pid", Value: newPID},
				{Key: "console_type", Value: machineType},
				{Key: "guid", Value: guid},
				{Key: "link_code", Value: database.GenerateLinkCode(10)},
			})

			if err = users.FindOne(nil, bson.M{"username": username}).Decode(&user); err != nil {
				log.Printf("Could not find newly-created user %s: %s\n", username, err)
				SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.OperationError)
				return
			}

		} else {
			// update console type and link code if needed for existing users
			updateFields := bson.D{}

			// always update console type to reflect current login platform
			if user.ConsoleType != machineType {
				log.Printf("Updating console type for %s from %d to %d\n", username, user.ConsoleType, machineType)
				updateFields = append(updateFields, bson.E{Key: "console_type", Value: machineType})
			}

			// generate link code if missing (which will be true for legacy accounts created before i made this system)
			if user.LinkCode == "" {
				linkCode := database.GenerateLinkCode(10)
				updateFields = append(updateFields, bson.E{Key: "link_code", Value: linkCode})
			}

			// only perform update if there are fields to update
			if len(updateFields) > 0 {
				_, err = users.UpdateOne(context.TODO(), bson.M{"username": username}, bson.D{
					{"$set", updateFields},
				})
				if err != nil {
					log.Printf("Could not update user data for %s: %s\n", username, err)
					SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.OperationError)
					return
				}
			}
		}
	case 2:
		// check if the machine ID is already in the DB
		machinesCollection := database.GocentralDatabase.Collection("machines")

		var machine models.Machine

		// try to find the machine via the Wii friend code (res[1])
		_ = machinesCollection.FindOne(context.TODO(), bson.M{"wii_friend_code": res[1]}).Decode(&machine)

		if machine.MachineID == 0 {
			log.Printf("Wii with friend code %v has never connected before - create DB entry\n", res[1])

			// race condition prevention
			newMachineID, err := database.GetNextMachineID(context.TODO())
			if err != nil {
				log.Printf("Could not get next machine ID: %s\n", err)
				SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.OperationError)
				return
			}

			_, err = machinesCollection.InsertOne(context.TODO(), bson.D{
				{Key: "wii_friend_code", Value: res[1]},
				{Key: "console_type", Value: 2},
				{Key: "machine_id", Value: newMachineID},
				{Key: "status", Value: ""},
			})

			if err != nil {
				log.Printf("Could not create Wii with friend code %v: %s\n", res[1], err)
				SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.OperationError)
				return
			}
		} else {
			user.PID = uint32(machine.MachineID)
			client.WiiFC = machine.WiiFriendCode
			log.Printf("Wii client detected with friend code %v, pid %v, username %v %v\n", client.WiiFC, user.PID, username, client.Username)
		}
	}

	client.Username = username

	var encryptedTicket []byte
	var kerberosKey []byte

	client.SetPlayerID(user.PID)

	log.Printf("%s requesting log in, has PID %v\n", username, user.PID)

	// generate the ticket and pass the friend code as the pwd on Wii, or use static password on PS3
	if machineType == 2 {
		encryptedTicket, kerberosKey = generateKerberosTicket(user.PID, uint32(serverPID), 16, client.WiiFC)
	} else {
		encryptedTicket, kerberosKey = generateKerberosTicket(user.PID, uint32(serverPID), 16, "")
	}
	mac := hmac.New(md5.New, kerberosKey)
	mac.Write(encryptedTicket)
	calculatedHmac := mac.Sum(nil)

	// Build the response body

	addr := os.Getenv("ADDRESS")

	if addr == "" {
		log.Println("ADDRESS is not set, clients will be unable to connect to the secure server. Please set the ADDRESS environment variable and restart GoCentral")
		SendErrorCode(AuthServer, client, nexproto.AuthenticationProtocolID, callID, quazal.AccessDenied)
		return
	}

	stationURL := fmt.Sprintf("prudps:/address=%s;port=%s;CID=1;PID=2;sid=1;stream=3;type=2", os.Getenv("ADDRESS"), os.Getenv("SECUREPORT"))

	rmcResponseStream := nex.NewStream()
	rmcResponseStream.Grow(int64(23))

	rmcResponseStream.WriteU32LENext([]uint32{0x10001}) // success
	rmcResponseStream.WriteU32LENext([]uint32{user.PID})
	rmcResponseStream.WriteBuffer(append(encryptedTicket[:], calculatedHmac[:]...))

	// RVConnectionData
	rmcResponseStream.WriteBufferString(stationURL) // Station
	rmcResponseStream.WriteU32LENext(([]uint32{0}))

	// dunno what this is looks like the response code again? not sure if its needed either but its on the end of real RB packets
	rmcResponseStream.WriteU32LENext([]uint32{0x1})
	rmcResponseStream.WriteU32LENext([]uint32{0x100})

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(nexproto.AuthenticationProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.AuthenticationMethodLogin, rmcResponseBody)

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
