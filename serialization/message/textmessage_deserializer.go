package message

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type TextMessageDeserializer struct{}

// writeUint32 appends a uint32 as little-endian bytes
func writeUint32(buf []byte, val uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, val)
	return append(buf, b...)
}

// writeUint64 appends a uint64 as little-endian bytes
func writeUint64(buf []byte, val uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, val)
	return append(buf, b...)
}

// writeBufferString appends a buffer string (uint32 length prefix + null-terminated string)
func writeBufferString(buf []byte, str string) []byte {
	strWithNull := str + "\x00"
	length := uint32(len(strWithNull))
	buf = writeUint32(buf, length)
	return append(buf, []byte(strWithNull)...)
}

// readBufferString reads a buffer string (uint32 length prefix + null-terminated string)
func readBufferString(data []byte, offset int) (string, int, error) {
	if len(data) < offset+4 {
		return "", offset, fmt.Errorf("insufficient data for buffer string length at offset %d", offset)
	}

	length := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	if len(data) < offset+int(length) {
		return "", offset, fmt.Errorf("insufficient data for buffer string content at offset %d, need %d bytes", offset, length)
	}

	str := string(data[offset : offset+int(length)])
	str = strings.TrimRight(str, "\x00")
	offset += int(length)

	return str, offset, nil
}

// Deserialize parses a TextMessage from raw bytes (wrapped in AnyDataHolder)
func (d *TextMessageDeserializer) Deserialize(data []byte) (TextMessage, error) {
	var msg TextMessage
	offset := 0

	// Parse AnyDataHolder wrapper
	// Type name (buffer string with uint32 length)
	typeName, newOffset, err := readBufferString(data, offset)
	if err != nil {
		return msg, fmt.Errorf("failed to read type name: %w", err)
	}
	offset = newOffset

	if typeName != "TextMessage" {
		return msg, fmt.Errorf("expected TextMessage type, got %s", typeName)
	}

	// Length including inner length field (uint32)
	if len(data) < offset+4 {
		return msg, fmt.Errorf("insufficient data for outer length at offset %d", offset)
	}
	offset += 4 // skip outer length

	// Inner length (uint32)
	if len(data) < offset+4 {
		return msg, fmt.Errorf("insufficient data for inner length at offset %d", offset)
	}
	offset += 4 // skip inner length

	// Now parse UserMessage fields
	if len(data) < offset+36 {
		return msg, fmt.Errorf("insufficient data for UserMessage fixed fields")
	}

	msg.ID = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.IDRecipient = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.RecipientType = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.ParentID = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.SenderPID = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.ReceptionTime.Value = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	msg.LifeTime = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	msg.Flags = binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Subject (NEX string with uint16 length)
	msg.Subject, offset, err = readBufferString(data, offset)
	if err != nil {
		return msg, fmt.Errorf("failed to read subject: %w", err)
	}

	// Sender (NEX string with uint16 length)
	msg.Sender, offset, err = readBufferString(data, offset)
	if err != nil {
		return msg, fmt.Errorf("failed to read sender: %w", err)
	}

	// TextMessage specific field: TextBody (NEX string with uint16 length)
	msg.TextBody, offset, err = readBufferString(data, offset)
	if err != nil {
		return msg, fmt.Errorf("failed to read text body: %w", err)
	}

	return msg, nil
}

// Serialize converts a TextMessage to bytes (wrapped in AnyDataHolder)
func (d *TextMessageDeserializer) Serialize(msg TextMessage) []byte {
	// Build the inner message content first
	var content []byte

	// UserMessage fields
	content = writeUint32(content, msg.ID)
	content = writeUint32(content, msg.IDRecipient)
	content = writeUint32(content, msg.RecipientType)
	content = writeUint32(content, msg.ParentID)
	content = writeUint32(content, msg.SenderPID)
	content = writeUint64(content, msg.ReceptionTime.Value)
	content = writeUint32(content, msg.LifeTime)
	content = writeUint32(content, msg.Flags)
	content = writeBufferString(content, msg.Subject)
	content = writeBufferString(content, msg.Sender)

	// TextMessage field
	content = writeBufferString(content, msg.TextBody)

	// Build the full output with AnyDataHolder wrapper
	var output []byte

	// Type name
	output = writeBufferString(output, "TextMessage")

	// Outer length (inner length field + content)
	outerLength := uint32(4 + len(content))
	output = writeUint32(output, outerLength)

	// Inner length
	output = writeUint32(output, uint32(len(content)))

	// Content
	output = append(output, content...)

	return output
}

// Serialize converts a UserMessage to bytes (wrapped in AnyDataHolder)
func (d *TextMessageDeserializer) SerializeUserMessage(msg UserMessage) []byte {
	// Build the inner message content first
	var content []byte
	// UserMessage fields
	content = writeUint32(content, msg.ID)
	content = writeUint32(content, msg.IDRecipient)
	content = writeUint32(content, msg.RecipientType)
	content = writeUint32(content, msg.ParentID)
	content = writeUint32(content, msg.SenderPID)
	content = writeUint64(content, msg.ReceptionTime.Value)
	content = writeUint32(content, msg.LifeTime)
	content = writeUint32(content, msg.Flags)
	content = writeBufferString(content, msg.Subject)
	content = writeBufferString(content, msg.Sender)
	// Build the full output with AnyDataHolder wrapper
	var output []byte
	// Type name
	output = writeBufferString(output, "UserMessage")
	// Outer length (inner length field + content)
	outerLength := uint32(4 + len(content))
	output = writeUint32(output, outerLength)
	// Inner length
	output = writeUint32(output, uint32(len(content)))
	// Content
	output = append(output, content...)
	return output
}
