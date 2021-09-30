package server

import (
	"fmt"
	"log"
	"sync"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type Router struct {
	server  *Wsps
	network sync.Map
	wan     *pkg.Wan
	lan     sync.Map
}

func (s *Wsps) NewRouter(ws *websocket.Conn) *Router {
	wan := pkg.NewWan(ws)
	return &Router{server: s, wan: wan}
}

func (r *Router) ServeConn() {
	for {
		_, data, err := r.wan.Read()
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			log.Println("error reading webSocket message:", err)
			break
		}
		var msg msg.WspMessage
		if proto.Unmarshal(data, &msg) != nil {
			log.Println("error unmarshal message:", err)
			continue
		}
		go r.process(&msg)
	}
	r.network.Range(func(key, value interface{}) bool {
		log.Printf("remove network %s", key)
		r.server.network.Delete(key)
		return true
	})
}

func (r *Router) process(message *msg.WspMessage) {
	switch message.Cmd {
	case msg.WspCmd_CONN_REQ:
		go r.NewConn(message)
	case msg.WspCmd_CONN_REP:
		r.Read(message)
	case msg.WspCmd_FORWARD:
		r.Read(message)
	case msg.WspCmd_CLOSE:
		if val, ok := r.lan.Load(message.Id); ok {
			r.lan.Delete(message.Id)
			val.(pkg.Bridge).Close()
		}
	}
}

func (r *Router) Read(message *msg.WspMessage) {
	if val, ok := r.lan.Load(message.Id); ok {
		if val.(pkg.Bridge).Read(message) {
			return
		}
		r.wan.CloseRemote(message.Id, "conn not exists")
	}
	r.wan.CloseRemote(message.Id, "conn not exists")
}

func (r *Router) NewConn(message *msg.WspMessage) {
	var addr msg.WspAddr
	err := proto.Unmarshal(message.Data, &addr)
	if err != nil {
		log.Println("error unmarshal addr", err)
		return
	}
	switch addr.Type {
	case msg.WspType_SOCKS5:
		r.NewSocks5Conn(message.Id, addr.Address)
	case msg.WspType_REMOTE:
		r.NewRemoteConn(message.Id, &addr)
	case msg.WspType_LOCAL:
		r.NewLocalConn(message.Id, &addr)
	default:
		r.wan.CloseRemote(message.Id, fmt.Sprintf("unkonwn %v", addr.Type))
	}
}

func (r *Router) NewLocalConn(id string, addr *msg.WspAddr) {
	if _, ok := r.server.network.Load(addr); ok {
		r.wan.CloseRemote(id, fmt.Sprintf("name %s registered", addr))
	} else {
		log.Printf("server register address %s", addr.Address)
		r.network.Store(addr.Address, addr)
		r.server.network.Store(addr.Address, r)
	}
}
