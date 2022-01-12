package server

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type Router struct {
	wsps    *Wsps
	hub     *Hub
	wan     *pkg.Wan
	routing *pkg.Routing
	http    sync.Map
}

func (wsps *Wsps) NewRouter(ws *websocket.Conn) *Router {
	wan := pkg.NewWan(ws)
	return &Router{wsps: wsps, wan: wan, hub: &Hub{}, routing: pkg.NewRouting()}
}

func (r *Router) GlobalHub() *Hub {
	return r.wsps.hub
}
func (r *Router) GlobalConfig() *Config {
	return r.wsps.config
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
		var m msg.WspMessage
		if proto.Unmarshal(data, &m) != nil {
			log.Println("error unmarshal message:", err)
			continue
		}
		r.process(&msg.Data{Msg: &m, Raw: &data})
	}
	r.GlobalHub().Delete(r.hub)
}

func (r *Router) process(data *msg.Data) {
	switch data.Cmd() {
	case msg.WspCmd_CONN_REQ:
		r.NewConn(data.Msg)
	default:
		err := r.routing.Routing(data)
		if errors.Is(err, pkg.ErrConnNotExist) {
			r.wan.CloseRemote(data.Id(), err.Error())
		}
	}
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
		go r.NewSocks5Conn(message.Id, addr.Address)
	case msg.WspType_REMOTE:
		r.NewRemoteConn(message.Id, &addr)
	case msg.WspType_LOCAL:
		r.AddLocalConn(message.Id, &addr)
	case msg.WspType_HTTP:
		r.AddHttpConn(message.Id, &addr)
	default:
		r.wan.CloseRemote(message.Id, fmt.Sprintf("unkonwn %v", addr.Type))
	}
}

func (r *Router) AddLocalConn(id string, addr *msg.WspAddr) {
	if r.GlobalHub().ExistHttp(addr.Address) {
		r.wan.CloseRemote(id, fmt.Sprintf("name %s registered", addr))
	} else {
		log.Printf("server register http %s", addr.Address)
		r.hub.AddLocal(addr.Address, addr)
		r.GlobalHub().AddLocal(addr.Address, r)
	}
}
