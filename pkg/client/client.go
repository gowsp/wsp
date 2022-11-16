package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/stream"
	"nhooyr.io/websocket"
)

type Wspc struct {
	start   sync.Once
	config  *Config
	listen  sync.Map
	wan     *stream.Wan
	handler *stream.Handler
}

func New(config *Config) *Wspc {
	w := &Wspc{config: config}
	w.handler = stream.NewHandler(w)
	return w
}
func (c *Wspc) register() {
	for _, val := range c.config.Remote {
		config, err := msg.NewWspConfig(msg.WspType_REMOTE, val)
		if err != nil {
			log.Println("forward remote error", err)
			continue
		}
		if _, err := c.wan.DialTCP(nil, config); err != nil {
			log.Println(err)
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
	headers := make(http.Header)
	headers.Set("Auth", c.config.Auth)
	headers.Set("Proto", msg.PROTOCOL_VERSION.String())
	server := c.config.Server
	ws, resp, err := websocket.Dial(context.Background(),
		server, &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		time.Sleep(3 * time.Second)
		log.Println("reconnect", server, "...")
		return c.connect()
	}
	if resp.StatusCode == 400 || resp.StatusCode == 401 {
		msg, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Fatalln(string(msg))
	}
	log.Println("successfully connected to", server)
	wan := stream.NewWan(ws)
	go wan.HeartBeat(time.Second * 30)
	return wan
}

func (c *Wspc) LoadConfig(channel string) (*msg.WspConfig, error) {
	if val, ok := c.listen.Load(channel); ok {
		return val.(*msg.WspConfig), nil
	}
	return nil, fmt.Errorf(channel + " not found")
}
