package rcenter

import "github.com/gorilla/websocket"

type UserConn struct {
	ws      *websocket.Conn
	lanAddr string
	lanPort string
	proto   string
}

type User struct {
	bus    *MessageBus
	name   string
	pass   string
	mainWs *websocket.Conn
	wsMap  map[string]*UserConn
}

func NewUser(bus *MessageBus, name string, pass string) *User {
	return &User{bus, name, pass, nil, make(map[string]*UserConn)}
}

func (user *User) requestControl(msg SeqMessage) *Message {
	user.bus.seqMsg <- msg
	return nil
}
