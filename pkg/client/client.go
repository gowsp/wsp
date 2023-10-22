package client

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/stream"
)

type Wspc struct {
	dialer  *ws.Dialer
	start   sync.Once
	config  WspcConfig
	listen  sync.Map
	wan     *stream.Wan
	handler *stream.Handler
}

func New(config WspcConfig) *Wspc {
	headers := make(http.Header)
	headers.Set("Auth", config.Auth)
	headers.Set("Proto", msg.PROTOCOL_VERSION.String())
	w := &Wspc{config: config,
		dialer: &ws.Dialer{Header: ws.HandshakeHeaderHTTP(headers)},
	}
	w.handler = stream.NewHandler(w)
	return w
}
func (c *Wspc) register() {
	for _, val := range c.config.Remote {
		config, err := msg.NewWspConfig(msg.WspType_REMOTE, val)
		if err != nil {
			logger.Error("remote config %s error: %s", val, err)
			continue
		}
		if _, err := c.wan.DialTCP(nil, config); err != nil {
			logger.Error("register %s error: %s", val, err)
		}
		c.listen.Store(config.Channel(), config)
	}
}
func (c *Wspc) ListenAndServe() {
	c.wan = c.connect()
	go c.register()
	c.start.Do(func() {
		c.LocalForward()
		c.DynamicForward()
	})
	c.handler.Serve(c.wan)
	c.ListenAndServe()
}
func (c *Wspc) connect() *stream.Wan {
	server := c.config.Server
	conn, _, _, err := c.dialer.Dial(context.Background(), server)
	if err != nil {
		time.Sleep(3 * time.Second)
		logger.Info("connect %s error %s, reconnect ...", server, err.Error())
		return c.connect()
	}
	logger.Info("successfully connected to %s", server)
	wan := stream.NewWan(conn, ws.StateClientSide)
	go wan.HeartBeat(time.Second * 30)
	return wan
}

func (c *Wspc) LoadConfig(channel string) (*msg.WspConfig, error) {
	if val, ok := c.listen.Load(channel); ok {
		return val.(*msg.WspConfig), nil
	}
	return nil, fmt.Errorf(channel + " not found")
}
