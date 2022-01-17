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

func (wsps *Wsps) NewRouter(ws *websocket.Conn) *Router {
	wan := proxy.NewWan(ws)
	return &Router{wsps: wsps, wan: wan, routing: proxy.NewRouting()}
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
			r.wan.ReplyMessage(data.Id(), false, err.Error())
		}
	default:
		err := r.routing.Routing(data)
		if errors.Is(err, proxy.ErrConnNotExist) {
			r.wan.CloseRemote(data.Id(), err.Error())
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
		return r.NewLocal(message.Id, conf)
	case msg.WspType_REMOTE:
		return r.AddRemote(message.Id, conf)
	default:
		return fmt.Errorf("unknown %v", req.Type)
	}
}

func (r *Router) AddRemote(id string, conf *msg.WspConfig) error {
	channel := conf.Channel()
	if r.wsps.Exist(channel) {
		return fmt.Errorf("channel %s already registered", conf.Channel())
	}
	if conf.IsHttp() {
		switch conf.Mode() {
		case "path":
			if conf.Value() == r.GlobalConfig().Path {
				return fmt.Errorf("setting the same path as wsps is not allowed")
			}
		case "domain":
			switch {
			case r.GlobalConfig().Host == "":
				return fmt.Errorf("wsps does not set host, domain method is not allowed")
			case r.GlobalConfig().Host == conf.Value():
				return fmt.Errorf("setting the same domain as wsps is not allowed")
			}
		default:
			return fmt.Errorf("unsupported http mode %s", conf.Mode())
		}
	}
	log.Println("register channel", channel)
	r.wan.Reply(id, true)
	r.wsps.Store(channel, r)
	r.channel.Store(channel, conf)
	return nil
}
