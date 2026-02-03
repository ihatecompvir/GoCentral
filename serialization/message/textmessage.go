package message

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type MessageRecipient struct {
	RecipientType uint32 // 1 = Principal ID, 2 = Gathering ID
	PrincipalID   uint32
	GatheringID   uint32
}

type UserMessage struct {
	ID            uint32
	IDRecipient   uint32
	RecipientType uint32
	ParentID      uint32
	SenderPID     uint32
	ReceptionTime DateTime
	LifeTime      uint32
	Flags         uint32
	Subject       string
	Sender        string
}

// TextMessage inherits from UserMessage and adds a text body
type TextMessage struct {
	UserMessage
	TextBody string
}

// MessageBodyData contains the parsed components of a TextMessage body
type MessageBodyData struct {
	RecipientType uint32
	GatheringID   uint32
	Message       string
}

// DateTime is a NEX DateTime format that packs date/time into a uint64 using bit fields:
// Bits 63-26: Year, Bits 25-22: Month, Bits 21-17: Day,
// Bits 16-12: Hour, Bits 11-6: Minute, Bits 5-0: Second
type DateTime struct {
	Value uint64
}

// NewDateTime creates a DateTime from a time.Time
func NewDateTime(t time.Time) DateTime {
	return DateTime{
		Value: (uint64(t.Year()) << 26) |
			(uint64(t.Month()) << 22) |
			(uint64(t.Day()) << 17) |
			(uint64(t.Hour()) << 12) |
			(uint64(t.Minute()) << 6) |
			uint64(t.Second()),
	}
}

// Now creates a DateTime from the current time
func DateTimeNow() DateTime {
	return NewDateTime(time.Now())
}

// Year extracts the year from the DateTime
func (d DateTime) Year() int {
	return int(d.Value >> 26)
}

// Month extracts the month from the DateTime
func (d DateTime) Month() int {
	return int((d.Value >> 22) & 0xF)
}

// Day extracts the day from the DateTime
func (d DateTime) Day() int {
	return int((d.Value >> 17) & 0x1F)
}

// Hour extracts the hour from the DateTime
func (d DateTime) Hour() int {
	return int((d.Value >> 12) & 0x1F)
}

// Minute extracts the minute from the DateTime
func (d DateTime) Minute() int {
	return int((d.Value >> 6) & 0x3F)
}

// Second extracts the second from the DateTime
func (d DateTime) Second() int {
	return int(d.Value & 0x3F)
}

// ToTime converts the DateTime to a time.Time
func (d DateTime) ToTime() time.Time {
	return time.Date(d.Year(), time.Month(d.Month()), d.Day(), d.Hour(), d.Minute(), d.Second(), 0, time.UTC)
}

// ParseBody parses the TextBody field which is in format "recipientType (guess, would make sense tho):gatheringID:message"
func (m *TextMessage) ParseBody() (MessageBodyData, error) {
	var data MessageBodyData

	parts := strings.SplitN(m.TextBody, ":", 3)
	if len(parts) < 3 {
		return data, fmt.Errorf("invalid message body format, expected 3 parts separated by colons, got %d", len(parts))
	}

	recipientType, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return data, fmt.Errorf("failed to parse recipient type: %w", err)
	}
	data.RecipientType = uint32(recipientType)

	gatheringID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return data, fmt.Errorf("failed to parse gathering ID: %w", err)
	}
	data.GatheringID = uint32(gatheringID)

	data.Message = parts[2]

	return data, nil
}
