package rcenter

import (
	"container/list"
	"fmt"
	"log"
	"time"
)

const (
	MESSAGE_SIZE    = 1024
	maxDuration     = time.Duration(15) * time.Minute
	expiredDuration = time.Duration(10) * time.Second
)

type MessageBus struct {
	timer       *time.Timer
	nextTick    time.Time
	expiredList *list.List
	seqMap      *SeqMap
	seqMsg      chan SeqMessage
	resp        chan *Message
}

func NewMessageBus() *MessageBus {
	bus := &MessageBus{
		expiredList: list.New(),
		seqMsg:      make(chan SeqMessage, MESSAGE_SIZE),
		resp:        make(chan *Message, MESSAGE_SIZE),
		seqMap:      NewSeqMap(MESSAGE_SIZE * 10),
	}
	return bus
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
