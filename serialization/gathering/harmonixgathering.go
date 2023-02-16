package serialization

// not sure what these fields are, but probably related to Overshell slots, instruments, and etc.
// would be good to eventually completely reverse this
type HmxGathering struct {
	Public byte
	Prop0  uint32
	Prop1  uint32
	Prop2  uint32
	Prop3  uint32
	Prop4  uint32
	Prop5  uint32
	Prop6  uint32
	Prop7  uint32
	Prop8  uint32
	Prop9  uint32
	Prop10 uint32
	Buffer uint32
}

type RVGathering struct {
	IDMyself            uint32       // the ID of the gathering
	IDOwner             uint32       // the PID of wh oowns the gathering
	IDHost              uint32       // the PID of the host
	MinParticipants     uint16       // always 0
	MaxParticipants     uint16       // always 4
	ParticipationPolicy uint32       // unknown, always 1
	PolicyArgument      uint32       // unknown
	Flags               uint32       // unknown
	State               uint32       // always 0, 2, or 6
	DescriptionCount    uint32       // unused, always a 1 byte null string
	DescriptionString   byte         // unused, always a 1 byte null string
	HarmonixGathering   HmxGathering // Harmonix-specific additions to the structure
}
