package server

import (
	"fmt"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
)

func (r *Router) AddRemote(id string, conf *msg.WspConfig) error {
	channel := conf.Channel()
	switch conf.Mode() {
	case "path":
		if conf.Value() == r.Config().Path {
			return fmt.Errorf("setting the same path as wsps is not allowed")
		}
	case "domain":
		switch {
		case r.Config().Host == "":
			return fmt.Errorf("wsps does not set host, domain method is not allowed")
		case r.Config().Host == conf.Value():
			return fmt.Errorf("setting the same domain as wsps is not allowed")
		}
	default:
		if conf.IsHTTP() {
			return fmt.Errorf("unsupported http mode %s", conf.Mode())
		}
	}
	if r.wsps.Exist(channel) {
		return fmt.Errorf("channel %s already registered", channel)
	}
	log.Println("register channel", channel)
	r.wan.Succeed(id)
	r.wsps.Store(channel, r)
	r.channel.Store(channel, conf)
	return nil
}

func (r *Router) NewRemoteConn(id string, conf *msg.WspConfig) error {
	key := conf.Channel()
	if val, ok := r.wsps.LoadRouter(key); ok {
		log.Printf("start bridge channel %s\n", key)
		remote := val.(*Router)

		_, ok := remote.channel.Load(key)
		if !ok {
			return fmt.Errorf("channel not registered")
		}
		remote.routing.AddPending(id, func(wm *msg.Data, res *msg.WspResponse) {
			if res.Code == msg.WspCode_SUCCESS {
				remoteWirter := proxy.NewWsRepeater(id, remote.wan)
				r.routing.AddRepeater(id, remoteWirter)
				localWriter := proxy.NewWsRepeater(id, r.wan)
				remote.routing.AddRepeater(id, localWriter)
			} else {
				remote.routing.DeleteConn(id)
			}
			r.wan.Write(*wm.Raw)
		})

		if err := remote.wan.Dail(id, conf); err != nil {
			remote.routing.DeleteConn(id)
			return err
		}
		return nil
	}
	return fmt.Errorf("channel not registered")
}
