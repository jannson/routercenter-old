package rcenter

import (
	"container/list"
	"time"
)

const (
	//	MessageEventOk      int = 0
	MessageEventErr     int = 1
	MessageEventTimeout int = 2
	MessageType
)

type Message struct {
	magic  uint32
	proto  uint8
	mType  uint8
	seq    uint16
	length uint32
	msg    []byte
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
	Fire(event int)
}

func (m *Message) GetRequestId() int {
	return int(m.seq)
}

func (m *Message) SetRequestId(seq int) {
	m.seq = uint16(seq)
}
