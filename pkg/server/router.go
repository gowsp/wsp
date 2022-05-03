package server

import (
	"fmt"
	"sync"

	"github.com/gowsp/wsp/pkg/channel"
	"github.com/gowsp/wsp/pkg/msg"
	"nhooyr.io/websocket"
)

type conn struct {
	wsps    *Wsps
	http    sync.Map
	configs sync.Map
	channel *channel.Channel
}

func (c *conn) config() *Config {
	return c.wsps.config
}

func (c *conn) ListenAndServe(ws *websocket.Conn) {
	c.channel = channel.New(ws, c.NewConn)
	c.channel.Serve()
	c.wsps.Delete(c.configs)
}

func (c *conn) NewConn(id string, req *msg.WspRequest) error {
	conf, err := req.ToConfig()
	if err != nil {
		return err
	}
	switch req.Type {
	case msg.WspType_DYNAMIC:
		return c.NewDynamic(id, conf)
	case msg.WspType_LOCAL:
		return c.NewRemoteConn(id, conf)
	case msg.WspType_REMOTE:
		return c.AddRemote(id, conf)
	default:
		return fmt.Errorf("unknown %v", req.Type)
	}
}
func (c *conn) LoadConfig(channel string) (*msg.WspConfig, error) {
	if val, ok := c.configs.Load(channel); ok {
		return val.(*msg.WspConfig), nil
	}
	return nil, fmt.Errorf(channel + " not found")
}
