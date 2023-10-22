package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/stream"
)

type conn struct {
	wsps   *Wsps
	wan    *stream.Wan
	http   sync.Map
	listen sync.Map
}

func (c *conn) config() *Config {
	return c.wsps.config
}

func (c *conn) close() {
	logger.Info("close websocket connect")
	c.listen.Range(func(key, value any) bool {
		hub.Remove(key.(string))
		return true
	})
}
func (c *conn) ListenAndServe(w net.Conn) {
	c.wan = stream.NewWan(w, ws.StateServerSide)
	stream.NewHandler(c).Serve(c.wan)
}

func (c *conn) NewConn(data *msg.Data, req *msg.WspRequest) error {
	conf, err := req.ToConfig()
	if err != nil {
		return err
	}
	id := data.ID()
	switch req.Type {
	case msg.WspType_DYNAMIC:
		return c.NewDynamic(id, conf)
	case msg.WspType_LOCAL:
		return c.NewRemoteConn(data, conf)
	case msg.WspType_REMOTE:
		return c.AddRemote(id, conf)
	default:
		return fmt.Errorf("unknown %v", req.Type)
	}
}
func (c *conn) LoadConfig(channel string) (*msg.WspConfig, error) {
	if val, ok := c.listen.Load(channel); ok {
		return val.(*msg.WspConfig), nil
	}
	return nil, fmt.Errorf(channel + " not found")
}
