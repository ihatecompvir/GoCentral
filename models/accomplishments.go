package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Accomplishments struct {
	ID                                   primitive.ObjectID `json:"_id" bson:"_id"`
	PID                                  int                `json:"pid" bson:"pid"`
	LBGoalValueCampaignMetascore         int                `json:"lb_goal_value_campaign_metascore" bson:"lb_goal_value_campaign_metascore"`
	LBGoalValueAccTourgoldlocal1         int                `json:"lb_goal_value_acc_tourgoldlocal1" bson:"lb_goal_value_acc_tourgoldlocal1"`
	LBGoalValueAccTourgoldlocal2         int                `json:"lb_goal_value_acc_tourgoldlocal2" bson:"lb_goal_value_acc_tourgoldlocal2"`
	LBGoalValueAccTourgoldregional1      int                `json:"lb_goal_value_acc_tourgoldregional1" bson:"lb_goal_value_acc_tourgoldregional1"`
	LBGoalValueAccTourgoldregional2      int                `json:"lb_goal_value_acc_tourgoldregional2" bson:"lb_goal_value_acc_tourgoldregional2"`
	LBGoalValueAccTourgoldcontinental1   int                `json:"lb_goal_value_acc_tourgoldcontinental1" bson:"lb_goal_value_acc_tourgoldcontinental1"`
	LBGoalValueAccTourgoldcontinental2   int                `json:"lb_goal_value_acc_tourgoldcontinental2" bson:"lb_goal_value_acc_tourgoldcontinental2"`
	LBGoalValueAccTourgoldcontinental3   int                `json:"lb_goal_value_acc_tourgoldcontinental3" bson:"lb_goal_value_acc_tourgoldcontinental3"`
	LBGoalValueAccTourgoldglobal1        int                `json:"lb_goal_value_acc_tourgoldglobal1" bson:"lb_goal_value_acc_tourgoldglobal1"`
	LBGoalValueAccTourgoldglobal2        int                `json:"lb_goal_value_acc_tourgoldglobal2" bson:"lb_goal_value_acc_tourgoldglobal2"`
	LBGoalValueAccTourgoldglobal3        int                `json:"lb_goal_value_acc_tourgoldglobal3" bson:"lb_goal_value_acc_tourgoldglobal3"`
	LBGoalValueAccOverdrivemaintain3     int                `json:"lb_goal_value_acc_overdrivemaintain3" bson:"lb_goal_value_acc_overdrivemaintain3"`
	LBGoalValueAccOverdrivecareer        int                `json:"lb_goal_value_acc_overdrivecareer" bson:"lb_goal_value_acc_overdrivecareer"`
	LBGoalValueAccCareersaves            int                `json:"lb_goal_value_acc_careersaves" bson:"lb_goal_value_acc_careersaves"`
	LBGoalValueAccMillionpoints          int                `json:"lb_goal_value_acc_millionpoints" bson:"lb_goal_value_acc_millionpoints"`
	LBGoalValueAccBassstreaklarge        int                `json:"lb_goal_value_acc_bassstreaklarge" bson:"lb_goal_value_acc_bassstreaklarge"`
	LBGoalValueAccHopothreehundredbass   int                `json:"lb_goal_value_acc_hopothreehundredbass" bson:"lb_goal_value_acc_hopothreehundredbass"`
	LBGoalValueAccDrumfill170            int                `json:"lb_goal_value_acc_drumfill170" bson:"lb_goal_value_acc_drumfill170"`
	LBGoalValueAccDrumstreaklong         int                `json:"lb_goal_value_acc_drumstreaklong" bson:"lb_goal_value_acc_drumstreaklong"`
	LBGoalValueAccDeployguitarfour       int                `json:"lb_goal_value_acc_deployguitarfour" bson:"lb_goal_value_acc_deployguitarfour"`
	LBGoalValueAccGuitarstreaklarge      int                `json:"lb_goal_value_acc_guitarstreaklarge" bson:"lb_goal_value_acc_guitarstreaklarge"`
	LBGoalValueAccHopoonethousand        int                `json:"lb_goal_value_acc_hopoonethousand" bson:"lb_goal_value_acc_hopoonethousand"`
	LBGoalValueAccDoubleawesomealot      int                `json:"lb_goal_value_acc_doubleawesomealot" bson:"lb_goal_value_acc_doubleawesomealot"`
	LBGoalValueAccTripleawesomealot      int                `json:"lb_goal_value_acc_tripleawesomealot" bson:"lb_goal_value_acc_tripleawesomealot"`
	LBGoalValueAccKeystreaklong          int                `json:"lb_goal_value_acc_keystreaklong" bson:"lb_goal_value_acc_keystreaklong"`
	LBGoalValueAccProbassstreakepic      int                `json:"lb_goal_value_acc_probassstreakepic" bson:"lb_goal_value_acc_probassstreakepic"`
	LBGoalValueAccProdrumroll3           int                `json:"lb_goal_value_acc_prodrumroll3" bson:"lb_goal_value_acc_prodrumroll3"`
	LBGoalValueAccProdrumstreaklong      int                `json:"lb_goal_value_acc_prodrumstreaklong" bson:"lb_goal_value_acc_prodrumstreaklong"`
	LBGoalValueAccProguitarstreakepic    int                `json:"lb_goal_value_acc_proguitarstreakepic" bson:"lb_goal_value_acc_proguitarstreakepic"`
	LBGoalValueAccProkeystreaklong       int                `json:"lb_goal_value_acc_prokeystreaklong" bson:"lb_goal_value_acc_prokeystreaklong"`
	LBGoalValueAccDeployvocals           int                `json:"lb_goal_value_acc_deployvocals" bson:"lb_goal_value_acc_deployvocals"`
	LBGoalValueAccDeployvocalsonehundred int                `json:"lb_goal_value_acc_deployvocalsonehundred" bson:"lb_goal_value_acc_deployvocalsonehundred"`
}
