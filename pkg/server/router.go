package server

import (
	"fmt"
	"sync"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"google.golang.org/protobuf/proto"
)

type Router struct {
	channel sync.Map
	http    sync.Map
	wsps    *Wsps
	wan     *proxy.Wan
	routing *proxy.Routing
}

func (r *Router) Wan() *proxy.Wan {
	return r.wan
}
func (r *Router) Routing() *proxy.Routing {
	return r.routing
}
func (r *Router) Config() *Config {
	return r.wsps.config
}

func (r *Router) Close() error {
	r.wsps.Delete(r.channel)
	return nil
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
