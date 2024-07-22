package setlists

import (
	"context"
	"fmt"
	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"math/rand"
	"rb3server/models"
	"rb3server/protocols/jsonproto/marshaler"
	"reflect"
)

type SetlistUpdateRequest struct {
	Region      string `json:"region"`
	SystemMS    int    `json:"system_ms"`
	MachineID   string `json:"machine_id"`
	SessionGUID string `json:"session_guid"`
	PID         int    `json:"pid"`
	Type        int    `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Flags       int    `json:"flags"`
	Shared      string `json:"shared"`
	ListGUID    string `json:"list_guid"`
	Art         string `json:"art"`
	SongID000   int    `json:"song_id000"`
	SongID001   int    `json:"song_id001"`
	SongID002   int    `json:"song_id002"`
	SongID003   int    `json:"song_id003"`
	SongID004   int    `json:"song_id004"`
	SongID005   int    `json:"song_id005"`
	SongID006   int    `json:"song_id006"`
	SongID007   int    `json:"song_id007"`
	SongID008   int    `json:"song_id008"`
	SongID009   int    `json:"song_id009"`
	SongID010   int    `json:"song_id010"`
	SongID011   int    `json:"song_id011"`
	SongID012   int    `json:"song_id012"`
	SongID013   int    `json:"song_id013"`
	SongID014   int    `json:"song_id014"`
	SongID015   int    `json:"song_id015"`
	SongID016   int    `json:"song_id016"`
	SongID017   int    `json:"song_id017"`
	SongID018   int    `json:"song_id018"`
	SongID019   int    `json:"song_id019"`
	SongID020   int    `json:"song_id020"`
	SongID021   int    `json:"song_id021"`
	SongID022   int    `json:"song_id022"`
	SongID023   int    `json:"song_id023"`
	SongID024   int    `json:"song_id024"`
	SongID025   int    `json:"song_id025"`
	SongID026   int    `json:"song_id026"`
	SongID027   int    `json:"song_id027"`
	SongID028   int    `json:"song_id028"`
	SongID029   int    `json:"song_id029"`
	SongID030   int    `json:"song_id030"`
	SongID031   int    `json:"song_id031"`
	SongID032   int    `json:"song_id032"`
	SongID033   int    `json:"song_id033"`
	SongID034   int    `json:"song_id034"`
	SongID035   int    `json:"song_id035"`
	SongID036   int    `json:"song_id036"`
	SongID037   int    `json:"song_id037"`
	SongID038   int    `json:"song_id038"`
	SongID039   int    `json:"song_id039"`
	SongID040   int    `json:"song_id040"`
	SongID041   int    `json:"song_id041"`
	SongID042   int    `json:"song_id042"`
	SongID043   int    `json:"song_id043"`
	SongID044   int    `json:"song_id044"`
	SongID045   int    `json:"song_id045"`
	SongID046   int    `json:"song_id046"`
	SongID047   int    `json:"song_id047"`
	SongID048   int    `json:"song_id048"`
	SongID049   int    `json:"song_id049"`
	SongID050   int    `json:"song_id050"`
	SongID051   int    `json:"song_id051"`
	SongID052   int    `json:"song_id052"`
	SongID053   int    `json:"song_id053"`
	SongID054   int    `json:"song_id054"`
	SongID055   int    `json:"song_id055"`
	SongID056   int    `json:"song_id056"`
	SongID057   int    `json:"song_id057"`
	SongID058   int    `json:"song_id058"`
	SongID059   int    `json:"song_id059"`
	SongID060   int    `json:"song_id060"`
	SongID061   int    `json:"song_id061"`
	SongID062   int    `json:"song_id062"`
	SongID063   int    `json:"song_id063"`
	SongID064   int    `json:"song_id064"`
	SongID065   int    `json:"song_id065"`
	SongID066   int    `json:"song_id066"`
	SongID067   int    `json:"song_id067"`
	SongID068   int    `json:"song_id068"`
	SongID069   int    `json:"song_id069"`
	SongID070   int    `json:"song_id070"`
	SongID071   int    `json:"song_id071"`
	SongID072   int    `json:"song_id072"`
	SongID073   int    `json:"song_id073"`
	SongID074   int    `json:"song_id074"`
	SongID075   int    `json:"song_id075"`
	SongID076   int    `json:"song_id076"`
	SongID077   int    `json:"song_id077"`
	SongID078   int    `json:"song_id078"`
	SongID079   int    `json:"song_id079"`
	SongID080   int    `json:"song_id080"`
	SongID081   int    `json:"song_id081"`
	SongID082   int    `json:"song_id082"`
	SongID083   int    `json:"song_id083"`
	SongID084   int    `json:"song_id084"`
	SongID085   int    `json:"song_id085"`
	SongID086   int    `json:"song_id086"`
	SongID087   int    `json:"song_id087"`
	SongID088   int    `json:"song_id088"`
	SongID089   int    `json:"song_id089"`
	SongID090   int    `json:"song_id090"`
	SongID091   int    `json:"song_id091"`
	SongID092   int    `json:"song_id092"`
	SongID093   int    `json:"song_id093"`
	SongID094   int    `json:"song_id094"`
	SongID095   int    `json:"song_id095"`
	SongID096   int    `json:"song_id096"`
	SongID097   int    `json:"song_id097"`
	SongID098   int    `json:"song_id098"`
	SongID099   int    `json:"song_id099"`
}

type SetlistUpdateResponse struct {
	Success int `json:"success"`
}

type SetlistUpdateService struct {
}

func (service SetlistUpdateService) Path() string {
	return "setlists/update"
}

func (service SetlistUpdateService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	var req SetlistUpdateRequest

	err := marshaler.UnmarshalRequest(data, &req)
	if err != nil {
		return "", err
	}

	if req.PID != int(client.PlayerID()) {
		log.Println("Client-supplied PID did not match server-assigned PID, rejecting request for setlist updates")
		return "", err
	}

	setlistCollection := database.Collection("setlists")
	usersCollection := database.Collection("users")
	var user models.User

	err = usersCollection.FindOne(nil, bson.M{"pid": req.PID}).Decode(&user)
	if err != nil {
		return "", err
	}

	SetlistID := rand.Intn(100000000)

	setlistDocument := bson.M{
		"id":         SetlistID,
		"type":       req.Type,
		"title":      req.Name,
		"desc":       req.Description,
		"flags":      req.Flags,
		"owner_guid": req.SessionGUID,
		"guid":       req.SessionGUID,
		"pid":        req.PID,
		"owner":      user.Username,
		"art":        req.Art,
	}
	reqValue := reflect.ValueOf(&req).Elem()
	for i := 0; i <= 99; i++ {
		fieldName := fmt.Sprintf("s_id%03d", i)
		songID := reqValue.FieldByName(fmt.Sprintf("SongID%03d", i)).Interface().(int)
		if songID != 0 {
			setlistDocument[fieldName] = songID
		}
	}

	err = setlistCollection.FindOne(context.TODO(), bson.M{"pid": req.PID, "title": req.Name}).Err()
	if err != nil { // If it's not nil document exists so we want to update to prevent duplicate entries
		filter := bson.M{"pid": req.PID, "title": req.Name}
		_, err = setlistCollection.UpdateOne(context.TODO(), filter, setlistDocument)
	} else {
		_, err = setlistCollection.InsertOne(context.TODO(), setlistDocument)
		if err != nil {
			return "", err
		}
	}

	res := []SetlistUpdateResponse{{1}}
	return marshaler.MarshalResponse(service.Path(), res)
}
