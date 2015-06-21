package rcenter

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"
)

const (
	MESSAGE_SIZE    = 1024
	maxDuration     = time.Duration(15) * time.Minute
	expiredDuration = time.Duration(10) * time.Second
)

type BusCall struct {
	Fn     interface{}
	Params []reflect.Value
	Result []reflect.Value
	Err    error
	Ok     chan bool
}

type MessageBus struct {
	users       map[string]*User
	timer       *time.Timer
	nextTick    time.Time
	expiredList *list.List
	seqMap      *SeqMap
	seqMsg      chan SeqMessage
	resp        chan *Message
	calls       chan *BusCall
}

func NewMessageBus() *MessageBus {
	bus := &MessageBus{
		users:       make(map[string]*User),
		expiredList: list.New(),
		seqMsg:      make(chan SeqMessage, MESSAGE_SIZE),
		resp:        make(chan *Message, MESSAGE_SIZE),
		seqMap:      NewSeqMap(MESSAGE_SIZE * 10),
		calls:       make(chan *BusCall),
	}
	return bus
}

func (bus *MessageBus) CheckLogin(u string, p string) bool {
	rv, err := bus.Call(bus.CheckLoginInner, u, p)
	if err != nil {
		return false
	}

	return rv[0].Bool()
}

func (bus *MessageBus) CheckLoginInner(u string, p string) bool {
	if userMgr, ok := bus.users[u]; ok {
		if userMgr.pass == p {
			return true
		}
	}

	return false
}

//Cannot block in bus thread!
func (bus *MessageBus) RequestControlInner(u string, deviceId string, msg *PMessage) {
	if userMgr, ok := bus.users[u]; ok {
		msg.processHandler = func(m *PMessage) error {
			if devConn, ok := userMgr.devMap[deviceId]; ok {
				devConn.writeMsg <- m.GetData().ToBytes()
				return nil
			} else {
				return errors.New("device not found")
			}
		}
		bus.seqMsg <- msg
	} else {
		//use it to return
		msg.resp <- errMessage
	}
}

func (bus *MessageBus) RequestControl(u string, deviceId string, msg *PMessage) *Message {
	bus.CallNoWait(bus.RequestControlInner, u, deviceId, msg)
	resp := <-msg.resp

	if resp.MType == PROTO_TYPE_ERROR {
		return nil
	}

	return resp
}

func (bus *MessageBus) Call(fn interface{}, params ...interface{}) (result []reflect.Value, err error) {
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	call := &BusCall{fn, in, nil, nil, make(chan bool)}
	bus.calls <- call

	//wait hear
	<-call.Ok

	close(call.Ok)
	return call.Result, call.Err
}

func (bus *MessageBus) CallNoWait(fn interface{}, params ...interface{}) {
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	call := &BusCall{fn, in, nil, nil, nil}
	bus.calls <- call
}

func (bus *MessageBus) Run() {
	l := bus.expiredList
	bus.nextTick = time.Now().Add(maxDuration)
	bus.timer = time.NewTimer(maxDuration)
	tCh := make(chan bool)

	go func() {
		for range bus.timer.C {
			tCh <- true
		}
	}()

	for {
		select {
		case call := <-bus.calls:
			f := reflect.ValueOf(call.Fn)
			if len(call.Params) != f.Type().NumIn() {
				call.Err = errors.New("The number of params is not adapted.")
				if call.Ok != nil {
					call.Ok <- false
				}
			} else {
				call.Result = f.Call(call.Params)
				if call.Ok != nil {
					call.Ok <- true
				}
			}

		case m := <-bus.seqMsg:
			oldData := bus.seqMap.NewSeq(m)
			if oldData != nil {
				fmt.Printf("FIXME: oldData is not null\n")
				oldData.(SeqMessage).Fire(MessageEventErr, nil)
			}
			now := time.Now()
			m.SetExpired(now.Add(expiredDuration))
			m.SetEl(l.PushBack(m))

			if bus.nextTick.After(m.GetExpred()) {
				bus.nextTick = m.GetExpred()
				bus.timer.Reset(bus.nextTick.Sub(now))
			}

			m.Fire(MessageEventOk, nil)

		case resp := <-bus.resp:
			data := bus.seqMap.GetData(int(resp.Seq))
			if data != nil {
				bus.seqMap.DelSeq(int(resp.Seq))
				seqMsg := data.(SeqMessage)
				seqMsg.PutResp(resp)
			} else {
				fmt.Printf("data not find in seqmap\n")
			}

		case <-tCh:
			now := time.Now()
			for l.Len() > 0 {
				el := l.Front()
				obj := el.Value.(SeqMessage)
				if obj.GetExpred().Before(now.Add(1 * time.Microsecond)) {
					//Timeout, remove from list
					l.Remove(el)
					obj.SetEl(nil)

					//remove from seqmap
					bus.seqMap.DelSeq(obj.GetRequestId())

					//Fire to response
					obj.(SeqMessage).Fire(MessageEventTimeout, nil)
				} else {
					//Not timeout
					bus.nextTick = l.Front().Value.(SeqMessage).GetExpred()
					bus.timer.Reset(bus.nextTick.Sub(now))
				}
			}

			if 0 == l.Len() {
				log.Println("all zero")
				bus.nextTick = time.Now().Add(maxDuration)
				bus.timer.Reset(bus.nextTick.Sub(now))
			}
		}
	}
}
