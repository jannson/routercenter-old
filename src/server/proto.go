package rcenter

import (
	"container/list"
	"time"
)

const (
	PROTO_MAGIC            uint32 = 0x10293874
	PROTO_PROTO_MAIN       uint8  = 0x1
	PROTO_TYPE_ERROR       uint8  = 0xFF
	PROTO_TYPE_TUNNEL_REQ  uint8  = 0x3
	PROTO_TYPE_TUNNEL_RESP uint8  = 0x4
)

var errMessage = &Message{
	MessageHeader{
		Magic:  PROTO_MAGIC,
		Proto:  0,
		Seq:    0,
		Unuse:  0,
		Length: 0,
		MType:  PROTO_TYPE_ERROR,
	},
	nil,
}

type PMessage struct {
	Message
	el             *list.Element
	expired        time.Time
	deviceId       string
	processHandler func(*PMessage) error
	resp           chan *Message
}

func NewPMessage(dev string, mtype uint8) *PMessage {
	p := &PMessage{el: nil, deviceId: dev, resp: make(chan *Message)}
	p.Magic = PROTO_MAGIC
	p.Proto = 1
	p.Unuse = 0
	p.Length = 0
	p.MType = mtype

	return p
}

func (m *PMessage) Release() {
	close(m.resp)
}

func (m *PMessage) PutResp(resp *Message) {
	m.resp <- resp
}

func (m *PMessage) GetData() *Message {
	return &m.Message
}

func (m *PMessage) SetExpired(expired time.Time) {
	m.expired = expired
}

func (m *PMessage) GetExpred() time.Time {
	return m.expired
}

func (m *PMessage) SetEl(el *list.Element) {
	m.el = el
}

func (m *PMessage) GetEl() *list.Element {
	return m.el
}

func (m *PMessage) Fire(event int, arg interface{}) {
	if event != MessageEventOk {
		m.resp <- errMessage
	} else {
		m.processHandler(m)
	}
}
