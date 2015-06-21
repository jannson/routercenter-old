package rcenter

import (
	"errors"

	"github.com/gorilla/websocket"
)

type UserConn struct {
	clientWs  *websocket.Conn
	tunnelWs  *websocket.Conn
	u         string
	key       int
	lanAddr   uint32
	lanPort   uint32
	proto     int
	clientMsg chan []byte
	tunnelMsg chan []byte
	isClosed  int32
}

type DeviceConn struct {
	u        string
	deviceId string
	mainWs   *websocket.Conn
	wsMap    map[int]*UserConn
	writeMsg chan []byte
	//automic
	isClosed int32
}

type User struct {
	bus       *MessageBus
	name      string
	pass      string
	sessionId int
	devMap    map[string]*DeviceConn
}

func NewUser(bus *MessageBus, name string, pass string) *User {
	return &User{bus, name, pass, make(map[string]*DeviceConn)}
}

func NewUserConn(bus *MessageBus, u string, d string, addr uint32, port uint32, ws *websocket.Conn) *UserConn {
	if userMgr, ok := bus.users[u]; ok {
		uc := &UserConn{
			clientWs:  ws,
			tunnelWs:  nil,
			u:         u,
			key:       userMgr.sessionId + 1,
			lanAddr:   addr,
			lanPort:   port,
			proto:     2,
			isClosed:  0,
			clientMsg: make(chan []byte, 100),
			tunnelMsg: make(chan []byte, 100),
		}
		userMgr.sessionId += 1
		return uc
	} else {
		return nil
	}
}

func (user *User) RegistDevice(dev *DeviceConn) {
	if old, ok := user.devMap[dev.deviceId]; ok {
		//TODO have to close more
		old.mainWs.Close()
	}

	user.devMap[dev.deviceId] = dev
}

func (user *User) UnregistDevice(deviceId string) error {
	if dev, ok := user.devMap[deviceId]; ok {
		dev.mainWs.Close()
		for k, v := range dev.wsMap {
			v.ws.Close()
			delete(dev.wsMap, k)
		}

		delete(user.devMap, deviceId)
		return nil
	} else {
		return errors.New("not found")
	}
}

func (user *User) RegistConn(conn *UserConn, deviceId string) error {
	if dev, ok := user.devMap[deviceId]; ok {
		if _, ok2 := dev.wsMap[conn.key]; ok2 {
			return errors.New("exists old conn")
		}
		dev.wsMap[conn.key] = conn
		return nil
	} else {
		return errors.New("device not found")
	}
}

func (user *User) UnregistConn(key int, deviceId string) error {
	if dev, ok := user.devMap[deviceId]; ok {
		if conn, ok2 := dev.wsMap[key]; ok2 {
			delete(dev.wsMap, key)
			conn.ws.Close()
			return nil
		}
	}

	return errors.New("conn not found")
}
