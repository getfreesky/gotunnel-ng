package main

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"reflect"
)

type SessionManager struct {
	*Actor
	Sessions map[int64]*Session
}

func NewSessionManager(delivery *Delivery) *SessionManager {
	manager := &SessionManager{
		Actor:    NewActor(),
		Sessions: make(map[int64]*Session),
	}
	manager.Recv(delivery.IncomingPacket, func(v reflect.Value) {
		packet := v.Interface().([]byte)
		reader := bytes.NewReader(packet)
		var id int64
		err := binary.Read(reader, binary.BigEndian, &id)
		if err != nil {
			return
		}
		var packetType uint8
		err = binary.Read(reader, binary.BigEndian, &packetType)
		if err != nil {
			return
		}
		switch packetType {
		case CONNECT:
			hostPortBs, err := ioutil.ReadAll(reader)
			if err != nil {
				return
			}
			session, err := NewIncomingSession(id, delivery, string(hostPortBs))
			if err != nil {
				return
			}
			manager.Sessions[id] = session
			session.OnClose(func() {
				delete(manager.Sessions, id)
			})
			manager.Signal("newSession", session)
		case DATA:
			data, err := ioutil.ReadAll(reader)
			if err != nil {
				return
			}
			session := manager.Sessions[id]
			if session == nil {
				return
			}
			session.Signal("data", data)
		case CLOSE:
			session := manager.Sessions[id]
			if session == nil {
				return
			}
			session.Signal("close")
		default:
			log.Fatal("unknown packet type %d", packetType)
		}
	})
	return manager
}
