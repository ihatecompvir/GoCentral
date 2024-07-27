package authentication

import (
	"bytes"
	"encoding/binary"
)

type NPTicket struct {

	// this is little endian
	Size1 uint32 // the size of the NPTicket
	Size2 uint32 // the size of the NPTicket again? (always seems to be the same as above)

	// everything after is big endian
	Version    uint8
	Unknown    uint8
	Unknown2   uint8
	Unknown3   uint8
	TicketSize uint32
	BodyType   uint16 // type as in datatype
	BodySize   uint16
	Body       []byte
	FooterType uint16
	FooterSize uint16
	Footer     NPTicketFooter
}

type NPTicketFooter struct {
	// everything is big endian
	CipherIDType  uint16 // type as in datatype
	CipherIDSize  uint16
	CipherID      uint32
	SignatureType uint16
	SignatureSize uint16
	Signature     []byte // asn.1 encoded
}

func (t *NPTicket) Bytes() []byte {
	size1Bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(size1Bytes, t.Size1)
	size2Bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(size2Bytes, t.Size2)
	ticketSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(ticketSizeBytes, t.TicketSize)
	bodyTypeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bodyTypeBytes, t.BodyType)
	bodySizeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bodySizeBytes, t.BodySize)
	footerTypeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(footerTypeBytes, t.FooterType)
	footerSizeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(footerSizeBytes, t.FooterSize)

	buf := bytes.Buffer{}
	buf.Write(size1Bytes)
	buf.Write(size2Bytes)
	buf.WriteByte(t.Version)
	buf.WriteByte(t.Unknown)
	buf.WriteByte(t.Unknown2)
	buf.WriteByte(t.Unknown3)
	buf.Write(ticketSizeBytes)
	buf.Write(bodyTypeBytes)
	buf.Write(bodySizeBytes)
	buf.Write(t.Body)
	buf.Write(footerTypeBytes)
	buf.Write(footerSizeBytes)
	footerBytes := t.Footer.Bytes()
	buf.Write(footerBytes)

	return buf.Bytes()
}

func (f *NPTicketFooter) Bytes() []byte {
	cipherIDTypeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(cipherIDTypeBytes, f.CipherIDType)
	cipherIDSizeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(cipherIDSizeBytes, f.CipherIDSize)
	cipherIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(cipherIDBytes, f.CipherID)
	signatureTypeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(signatureTypeBytes, f.SignatureType)
	signatureSizeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(signatureSizeBytes, f.SignatureSize)

	buf := bytes.Buffer{}
	buf.Write(cipherIDTypeBytes)
	buf.Write(cipherIDSizeBytes)
	buf.Write(cipherIDBytes)
	buf.Write(signatureTypeBytes)
	buf.Write(signatureSizeBytes)
	buf.Write(f.Signature)

	return buf.Bytes()
}

type NPTicketDeserializer struct{}

func (d *NPTicketDeserializer) Deserialize(data []byte) (NPTicket, error) {
	ticket := &NPTicket{}
	ticket.Size1 = binary.LittleEndian.Uint32(data[:4])
	ticket.Size2 = binary.LittleEndian.Uint32(data[4:8])
	ticket.Version = uint8(data[8])
	ticket.Unknown = uint8(data[9])
	ticket.Unknown2 = uint8(data[10])
	ticket.Unknown3 = uint8(data[11])
	ticket.TicketSize = binary.BigEndian.Uint32(data[12:16])
	ticket.BodyType = binary.BigEndian.Uint16(data[16:18])
	ticket.BodySize = binary.BigEndian.Uint16(data[18:20])
	ticket.Body = data[20 : 20+ticket.BodySize]
	footerStart := 20 + ticket.BodySize
	ticket.FooterType = binary.BigEndian.Uint16(data[footerStart : footerStart+2])
	ticket.FooterSize = binary.BigEndian.Uint16(data[footerStart+2 : footerStart+4])
	footerData := data[footerStart+4 : footerStart+4+ticket.FooterSize]
	footer := &NPTicketFooter{}
	footer.CipherIDType = binary.BigEndian.Uint16(footerData[:2])
	footer.CipherIDSize = binary.BigEndian.Uint16(footerData[2:4])
	footer.CipherID = binary.BigEndian.Uint32(footerData[4:8])
	footer.SignatureType = binary.BigEndian.Uint16(footerData[8:10])
	footer.SignatureSize = binary.BigEndian.Uint16(footerData[10:12])
	footer.Signature = footerData[12 : 12+footer.SignatureSize]
	ticket.Footer = *footer
	return *ticket, nil
}
