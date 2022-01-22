package server

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type Router struct {
	channel sync.Map
	http    sync.Map
	wsps    *Wsps
	wan     *proxy.Wan
	routing *proxy.Routing
}

func (s *Wsps) NewRouter(ws *websocket.Conn) *Router {
	wan := proxy.NewWan(ws)
	return &Router{wsps: s, wan: wan, routing: proxy.NewRouting()}
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
	r.wsps.Delete(r.channel)
}

func (r *Router) process(data *msg.Data) {
	switch data.Cmd() {
	case msg.WspCmd_CONNECT:
		err := r.NewConn(data.Msg)
		if err != nil {
			r.wan.ReplyMessage(data.ID(), false, err.Error())
		}
	default:
		err := r.routing.Routing(data)
		if errors.Is(err, proxy.ErrConnNotExist) {
			r.wan.CloseRemote(data.ID(), err.Error())
		}
	}
}

func (r *Router) NewConn(message *msg.WspMessage) error {
	var req msg.WspRequest
	err := proto.Unmarshal(message.Data, &req)
	if err != nil {
		return err
	}
	conf, err := req.ToConfig()
	if err != nil {
		return err
	}
	switch req.Type {
	case msg.WspType_DYNAMIC:
		go r.NewDynamic(message.Id, conf)
		return nil
	case msg.WspType_LOCAL:
		return r.NewRemoteConn(message.Id, conf)
	case msg.WspType_REMOTE:
		return r.AddRemote(message.Id, conf)
	default:
		return fmt.Errorf("unknown %v", req.Type)
	}
}
