package models

type AccomplishmentScoreEntry struct {
	PID   int `json:"pid" bson:"pid"`
	Score int `json:"score" bson:"score"`
}

type Accomplishments struct {
	LBGoalValueCampaignMetascore         []AccomplishmentScoreEntry `json:"lb_goal_value_campaign_metascore" bson:"lb_goal_value_campaign_metascore"`
	LBGoalValueAccTourgoldlocal1         []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldlocal1" bson:"lb_goal_value_acc_tourgoldlocal1"`
	LBGoalValueAccTourgoldlocal2         []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldlocal2" bson:"lb_goal_value_acc_tourgoldlocal2"`
	LBGoalValueAccTourgoldregional1      []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldregional1" bson:"lb_goal_value_acc_tourgoldregional1"`
	LBGoalValueAccTourgoldregional2      []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldregional2" bson:"lb_goal_value_acc_tourgoldregional2"`
	LBGoalValueAccTourgoldcontinental1   []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldcontinental1" bson:"lb_goal_value_acc_tourgoldcontinental1"`
	LBGoalValueAccTourgoldcontinental2   []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldcontinental2" bson:"lb_goal_value_acc_tourgoldcontinental2"`
	LBGoalValueAccTourgoldcontinental3   []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldcontinental3" bson:"lb_goal_value_acc_tourgoldcontinental3"`
	LBGoalValueAccTourgoldglobal1        []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldglobal1" bson:"lb_goal_value_acc_tourgoldglobal1"`
	LBGoalValueAccTourgoldglobal2        []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldglobal2" bson:"lb_goal_value_acc_tourgoldglobal2"`
	LBGoalValueAccTourgoldglobal3        []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tourgoldglobal3" bson:"lb_goal_value_acc_tourgoldglobal3"`
	LBGoalValueAccOverdrivemaintain3     []AccomplishmentScoreEntry `json:"lb_goal_value_acc_overdrivemaintain3" bson:"lb_goal_value_acc_overdrivemaintain3"`
	LBGoalValueAccOverdrivecareer        []AccomplishmentScoreEntry `json:"lb_goal_value_acc_overdrivecareer" bson:"lb_goal_value_acc_overdrivecareer"`
	LBGoalValueAccCareersaves            []AccomplishmentScoreEntry `json:"lb_goal_value_acc_careersaves" bson:"lb_goal_value_acc_careersaves"`
	LBGoalValueAccMillionpoints          []AccomplishmentScoreEntry `json:"lb_goal_value_acc_millionpoints" bson:"lb_goal_value_acc_millionpoints"`
	LBGoalValueAccBassstreaklarge        []AccomplishmentScoreEntry `json:"lb_goal_value_acc_bassstreaklarge" bson:"lb_goal_value_acc_bassstreaklarge"`
	LBGoalValueAccHopothreehundredbass   []AccomplishmentScoreEntry `json:"lb_goal_value_acc_hopothreehundredbass" bson:"lb_goal_value_acc_hopothreehundredbass"`
	LBGoalValueAccDrumfill170            []AccomplishmentScoreEntry `json:"lb_goal_value_acc_drumfill170" bson:"lb_goal_value_acc_drumfill170"`
	LBGoalValueAccDrumstreaklong         []AccomplishmentScoreEntry `json:"lb_goal_value_acc_drumstreaklong" bson:"lb_goal_value_acc_drumstreaklong"`
	LBGoalValueAccDeployguitarfour       []AccomplishmentScoreEntry `json:"lb_goal_value_acc_deployguitarfour" bson:"lb_goal_value_acc_deployguitarfour"`
	LBGoalValueAccGuitarstreaklarge      []AccomplishmentScoreEntry `json:"lb_goal_value_acc_guitarstreaklarge" bson:"lb_goal_value_acc_guitarstreaklarge"`
	LBGoalValueAccHopoonethousand        []AccomplishmentScoreEntry `json:"lb_goal_value_acc_hopoonethousand" bson:"lb_goal_value_acc_hopoonethousand"`
	LBGoalValueAccDoubleawesomealot      []AccomplishmentScoreEntry `json:"lb_goal_value_acc_doubleawesomealot" bson:"lb_goal_value_acc_doubleawesomealot"`
	LBGoalValueAccTripleawesomealot      []AccomplishmentScoreEntry `json:"lb_goal_value_acc_tripleawesomealot" bson:"lb_goal_value_acc_tripleawesomealot"`
	LBGoalValueAccKeystreaklong          []AccomplishmentScoreEntry `json:"lb_goal_value_acc_keystreaklong" bson:"lb_goal_value_acc_keystreaklong"`
	LBGoalValueAccProbassstreakepic      []AccomplishmentScoreEntry `json:"lb_goal_value_acc_probassstreakepic" bson:"lb_goal_value_acc_probassstreakepic"`
	LBGoalValueAccProdrumroll3           []AccomplishmentScoreEntry `json:"lb_goal_value_acc_prodrumroll3" bson:"lb_goal_value_acc_prodrumroll3"`
	LBGoalValueAccProdrumstreaklong      []AccomplishmentScoreEntry `json:"lb_goal_value_acc_prodrumstreaklong" bson:"lb_goal_value_acc_prodrumstreaklong"`
	LBGoalValueAccProguitarstreakepic    []AccomplishmentScoreEntry `json:"lb_goal_value_acc_proguitarstreakepic" bson:"lb_goal_value_acc_proguitarstreakepic"`
	LBGoalValueAccProkeystreaklong       []AccomplishmentScoreEntry `json:"lb_goal_value_acc_prokeystreaklong" bson:"lb_goal_value_acc_prokeystreaklong"`
	LBGoalValueAccDeployvocals           []AccomplishmentScoreEntry `json:"lb_goal_value_acc_deployvocals" bson:"lb_goal_value_acc_deployvocals"`
	LBGoalValueAccDeployvocalsonehundred []AccomplishmentScoreEntry `json:"lb_goal_value_acc_deployvocalsonehundred" bson:"lb_goal_value_acc_deployvocalsonehundred"`
}
