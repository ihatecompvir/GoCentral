package serialization

import (
	"encoding/binary"
	"fmt"
)

type GatheringDeserializer struct{}

// TODO: Redo this to use Go's marshaling because that would probably be better
func (d *GatheringDeserializer) Deserialize(data []byte) (RVGathering, error) {
	if len(data) < 54 {
		return RVGathering{}, fmt.Errorf("insufficient data to deserialize Gathering, expected at least 54 bytes but got %d", len(data))
	}

	var g RVGathering

	// Read fixed-size fields
	g.IDMyself = binary.LittleEndian.Uint32(data[0:4])
	g.IDOwner = binary.LittleEndian.Uint32(data[4:8])
	g.IDHost = binary.LittleEndian.Uint32(data[8:12])
	g.MinParticipants = binary.LittleEndian.Uint16(data[12:14])
	g.MaxParticipants = binary.LittleEndian.Uint16(data[14:16])
	g.ParticipationPolicy = binary.LittleEndian.Uint32(data[16:20])
	g.PolicyArgument = binary.LittleEndian.Uint32(data[20:24])
	g.Flags = binary.LittleEndian.Uint32(data[24:28])
	g.State = binary.LittleEndian.Uint32(data[28:32])
	g.DescriptionCount = binary.LittleEndian.Uint32(data[32:36])
	g.DescriptionString = data[36]

	// Read HarmonixGathering struct
	var h HmxGathering
	h.Public = data[37]
	h.Prop0 = binary.LittleEndian.Uint32(data[38:42])
	h.Prop1 = binary.LittleEndian.Uint32(data[42:46])
	h.Prop2 = binary.LittleEndian.Uint32(data[46:50])
	h.Prop3 = binary.LittleEndian.Uint32(data[50:54])
	h.Prop4 = binary.LittleEndian.Uint32(data[54:58])
	h.Prop5 = binary.LittleEndian.Uint32(data[58:62])
	h.Prop6 = binary.LittleEndian.Uint32(data[62:66])
	h.Prop7 = binary.LittleEndian.Uint32(data[66:70])
	h.Prop8 = binary.LittleEndian.Uint32(data[70:74])
	h.Prop9 = binary.LittleEndian.Uint32(data[74:78])
	h.Prop10 = binary.LittleEndian.Uint32(data[78:82])
	h.Buffer = binary.LittleEndian.Uint32(data[82:86])

	g.HarmonixGathering = h

	return g, nil
}
