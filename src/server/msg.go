package rcenter

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/binary"
	"time"
)

const (
	MessageEventOk      int = 0
	MessageEventErr     int = 1
	MessageEventTimeout int = 2
	MessageType
)

type MessageHeader struct {
	Magic  uint32
	Proto  uint8
	MType  uint8
	Seq    uint16
	Unuse  uint32
	Length uint32
}

//TODO do better hear
type MsgHandshake struct {
	Ulen     int32
	Plen     int32
	Dlen     int32
	Username [32]byte
	Pass     [32]byte
	DeviceId [32]byte
}

type MsgTunnel struct {
	lanAddr uint32
	lanPort uint32
}

type Message struct {
	MessageHeader

	msg []byte
}

type SeqMessage interface {
	GetRequestId() int
	SetRequestId(seq int)
	PutResp(m *Message)
	GetData() *Message
	SetExpired(expired time.Time)
	GetExpred() time.Time
	SetEl(*list.Element)
	GetEl() *list.Element
	Fire(event int, arg interface{})
}

func (m *Message) GetRequestId() int {
	return int(m.Seq)
}

func (m *Message) SetRequestId(seq int) {
	m.Seq = uint16(seq)
}

func (m *Message) ToBytes() []byte {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	binary.Write(writer, binary.BigEndian, &m.MessageHeader)
	binary.Write(writer, binary.BigEndian, m.msg)
	writer.Flush()
	return buf.Bytes()
}
