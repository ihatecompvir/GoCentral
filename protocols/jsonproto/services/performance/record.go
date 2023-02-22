package performance

import (
	"rb3server/protocols/jsonproto/marshaler"

	"github.com/ihatecompvir/nex-go"
	"go.mongodb.org/mongo-driver/mongo"
)

type PerformanceRecordRequest struct {
	Region                                     string  `json:"region"`
	Mode                                       string  `json:"mode"`
	Difficulty                                 int     `json:"difficulty"`
	NumVocalParts                              int     `json:"num_vocal_parts"`
	Stars                                      int     `json:"stars"`
	SystemMs                                   int     `json:"system_ms"`
	NotesHitFraction                           float32 `json:"notes_hit_fraction"`
	SongID                                     int     `json:"song_id"`
	IsOnline                                   int     `json:"is_online"`
	ScoreType                                  int     `json:"score_type"`
	MachineID                                  string  `json:"machine_id"`
	SessionGUID                                string  `json:"session_guid"`
	PID                                        int     `json:"pid"`
	IsPlaytest                                 int     `json:"is_playtest"`
	IsCheating                                 int     `json:"is_cheating"`
	TimeStamp                                  int     `json:"time_stamp"`
	EndGameScore                               int     `json:"end_game_score"`
	HitCount                                   int     `json:"hit_count"`
	MissCount                                  int     `json:"miss_count"`
	TimesSaved                                 int     `json:"times_saved"`
	PlayersSaved                               int     `json:"players_saved"`
	HitStreakStart                             int     `json:"hit_streak_start"`
	HitStreakDuration                          int     `json:"hit_streak_duration"`
	EndGameOverdrive                           float32 `json:"end_game_overdrive"`
	EndGameCrowdLevel                          float32 `json:"end_game_crowd_level"`
	CodaPoints                                 int     `json:"coda_points"`
	ODPhrasesCompleted                         int     `json:"od_phrases_completed"`
	ODPhrasesCount                             int     `json:"od_phrases_count"`
	UnisonPhrasesCompleted                     int     `json:"unison_phrases_completed"`
	UnisonPhrasesCount                         int     `json:"unison_phrases_count"`
	AverageMSError                             float32 `json:"average_ms_error"`
	FailurePoint000                            float32 `json:"failure_point000"`
	FailurePoint001                            float32 `json:"failure_point001"`
	FailurePoint002                            float32 `json:"failure_point002"`
	SavePoint000                               float32 `json:"save_point000"`
	SavePoint001                               float32 `json:"save_point001"`
	SavePoint002                               float32 `json:"save_point002"`
	TimesSaved000                              float32 `json:"times_saved000"`
	TimesSaved001                              float32 `json:"times_saved001"`
	TimesSaved002                              float32 `json:"times_saved002"`
	PlayersSaved000                            float32 `json:"players_saved000"`
	PlayersSaved001                            float32 `json:"players_saved001"`
	PlayersSaved002                            float32 `json:"players_saved002"`
	BestSolo000                                int     `json:"best_solo000"`
	BestSolo001                                int     `json:"best_solo001"`
	BestSolo002                                int     `json:"best_solo002"`
	HitStreakCount                             int     `json:"hit_streak_count"`
	HitStreakStart000                          int     `json:"hit_streak_start000"`
	HitStreakDuration000                       int     `json:"hit_streak_duration000"`
	HitStreakStart001                          int     `json:"hit_streak_start001"`
	HitStreakDuration001                       int     `json:"hit_streak_duration001"`
	HitStreakStart002                          int     `json:"hit_streak_start002"`
	HitStreakDuration002                       int     `json:"hit_streak_duration002"`
	MissStreakCount                            int     `json:"miss_streak_count"`
	MissStreakStart000                         int     `json:"miss_streak_start000"`
	MissStreakDuration000                      int     `json:"miss_streak_duration000"`
	MissStreakStart001                         int     `json:"miss_streak_start001"`
	MissStreakDuration001                      int     `json:"miss_streak_duration001"`
	MissStreakStart002                         int     `json:"miss_streak_start002"`
	MissStreakDuration002                      int     `json:"miss_streak_duration002"`
	BestODDeploymentCount                      int     `json:"best_od_deployment_count"`
	BestODDeploymentStart000                   int     `json:"best_od_deployment_start000"`
	BestODDeploymentDuration000                int     `json:"best_od_deployment_duration000"`
	BestODDeploymentStartingMultiplier000      int     `json:"best_od_deployment_starting_multiplier000"`
	BestODDeploymentEndingMultiplier000        int     `json:"best_od_deployment_ending_multiplier000"`
	BestODDeploymentPoints000                  int     `json:"best_od_deployment_points000"`
	BestODDeploymentStart001                   int     `json:"best_od_deployment_start001"`
	BestODDeploymentDuration001                int     `json:"best_od_deployment_duration001"`
	BestODDeploymentStartingMultiplier001      int     `json:"best_od_deployment_starting_multiplier001"`
	BestODDeploymentEndingMultiplier001        int     `json:"best_od_deployment_ending_multiplier001"`
	BestODDeploymentPoints001                  int     `json:"best_od_deployment_points001"`
	BestODDeploymentStart002                   int     `json:"best_od_deployment_start002"`
	BestODDeploymentDuration002                int     `json:"best_od_deployment_duration002"`
	BestODDeploymentStartingMultiplier002      int     `json:"best_od_deployment_starting_multiplier002"`
	BestODDeploymentEndingMultiplier002        int     `json:"best_od_deployment_ending_multiplier002"`
	BestODDeploymentPoints002                  int     `json:"best_od_deployment_points002"`
	BestStreakMultipliersCount                 int     `json:"best_streak_multipliers_count"`
	BestStreakMultipliersStart000              int     `json:"best_streak_multiplier_start000"`
	BestStreakMultipliersDuration000           int     `json:"best_streak_multiplier_duration000"`
	BestStreakMultipliersStartingMultiplier000 int     `json:"best_streak_multiplier_starting_multiplier000"`
	BestStreakMultipliersEndingMultiplier000   int     `json:"best_streak_multiplier_ending_multiplier000"`
	BestStreakMultipliersPoints000             int     `json:"best_streak_multiplier_points000"`
	BestStreakMultipliersStart001              int     `json:"best_streak_multiplier_start001"`
	BestStreakMultipliersDuration001           int     `json:"best_streak_multiplier_duration001"`
	BestStreakMultipliersStartingMultiplier001 int     `json:"best_streak_multiplier_starting_multiplier001"`
	BestStreakMultipliersEndingMultiplier001   int     `json:"best_streak_multiplier_ending_multiplier001"`
	BestStreakMultipliersPoints001             int     `json:"best_streak_multiplier_points001"`
	BestStreakMultipliersStart002              int     `json:"best_streak_multiplier_start002"`
	BestStreakMultipliersDuration002           int     `json:"best_streak_multiplier_duration002"`
	BestStreakMultipliersStartingMultiplier002 int     `json:"best_streak_multiplier_starting_multiplier002"`
	BestStreakMultipliersEndingMultiplier002   int     `json:"best_streak_multiplier_ending_multiplier002"`
	BestStreakMultipliersPoints002             int     `json:"best_streak_multiplier_points002"`
	TotalODDuration                            int     `json:"total_od_duration"`
	TotalMultiplierDuration                    int     `json:"total_multiplier_duration"`
	RollsHitCompletely                         int     `json:"rolls_hit_completely"`
	RollCount                                  int     `json:"roll_count"`
	HopoGemsHopoed                             int     `json:"hopo_gems_hopoed"`
	HopoGemCount                               int     `json:"hopo_gem_count"`
	HighGemsHitHigh                            int     `json:"high_gems_hit_high"`
	HighGemsHitLow                             int     `json:"high_gems_hit_low"`
	HighFretGemCount                           int     `json:"high_fret_gem_count"`
	SustainGemsHitCompletely                   int     `json:"sustain_gems_hit_completely"`
	SustainGemsHitPartially                    int     `json:"sustain_gems_hit_partially"`
	SustainGemsCount                           int     `json:"sustain_gems_count"`
	TrillsHitCompletely                        int     `json:"trills_hit_completely"`
	TrillsHitPartially                         int     `json:"trills_gems_hit_partially"`
	TrillCount                                 int     `json:"trill_count"`
	CymbalGemsHitOnCymbals                     int     `json:"cymbal_gems_hit_on_cymbals"`
	CymbalGemsHitOnPads                        int     `json:"cymbal_gems_hit_on_pads"`
	CymbalGemCount                             int     `json:"cymbal_gem_count"`
	DoubleHarmonyHit                           int     `json:"double_harmony_hit"`
	DoubleHarmonyPhraseCount                   int     `json:"double_harmony_phrase_count"`
	TripleHarmonyHit                           int     `json:"triple_harmony_hit"`
	TripleHarmonyPhraseCount                   int     `json:"triple_harmony_phrase_count"`
	NumSingers                                 int     `json:"num_singers"`
	Singer000Part000Part                       int     `json:"singer000_part000_part"`
	Singer000Part000Pct                        float32 `json:"singer000_part000_pct"`
	Singer000PitchDeviation                    float32 `json:"singer000_pitch_deviation"`
	Singer000PitchDeviationOfDeviation         float32 `json:"singer000_pitch_deviation_of_deviation"`
	Singer001Part000Part                       int     `json:"singer001_part000_part"`
	Singer001Part000Pct                        float32 `json:"singer001_part000_pct"`
	Singer001PitchDeviation                    float32 `json:"singer001_pitch_deviation"`
	Singer001PitchDeviationOfDeviation         float32 `json:"singer001_pitch_deviation_of_deviation"`
	Singer002Part000Part                       int     `json:"singer002_part000_part"`
	Singer002Part000Pct                        float32 `json:"singer002_part000_pct"`
	Singer002PitchDeviation                    float32 `json:"singer002_pitch_deviation"`
	Singer002PitchDeviationOfDeviation         float32 `json:"singer002_pitch_deviation_of_deviation"`
}

type PerformanceRecordResponse struct {
	Test int `json:"test"`
}

type PerformanceRecordService struct {
}

func (service PerformanceRecordService) Path() string {
	return "performance/record"
}

func (service PerformanceRecordService) Handle(data string, database *mongo.Database, client *nex.Client) (string, error) {
	//var req PerformanceRecordRequest
	//err := marshaler.UnmarshalRequest(data, &req)
	//if err != nil {
	//	return "", err
	//}

	// Spoof account linking status, 12345 pid
	res := []PerformanceRecordResponse{{
		1,
	}}

	return marshaler.MarshalResponse(service.Path(), res)
}
