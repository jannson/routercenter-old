package rcenter

import (
	"container/list"
	"time"
)

const (
	PROTO_MAGIC      uint32 = 0x10293874
	PROTO_PROTO_MAIN uint8  = 0x1
	PROTO_TYPE_ERROR uint8  = 0xFF
)

var errMessage = &Message{
	mType: PROTO_TYPE_ERROR,
}

type PMessage struct {
	Message
	el       *list.Element
	expired  time.Time
	deviceId string
	resp     chan *Message
}

func NewPMessage(dev string) *PMessage {
	return &PMessage{el: nil, deviceId: dev, resp: make(chan *Message)}
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

func (m *PMessage) Fire(event int) {
	m.resp <- errMessage
}
