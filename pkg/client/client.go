package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gowsp/wsp/pkg/channel"
	"github.com/gowsp/wsp/pkg/msg"
	"nhooyr.io/websocket"
)

type Wspc struct {
	start   sync.Once
	config  *Config
	closed  uint32
	configs sync.Map
	channel *channel.Channel
}

func New(config *Config) *Wspc {
	return &Wspc{config: config}
}

func (c *Wspc) ListenAndServe() {
	channel := c.connect()
	c.channel = channel
	c.Register()
	c.start.Do(func() {
		c.LocalForward()
		c.DynamicForward()
	})
	channel.Serve()
	c.ListenAndServe()
}
func (c *Wspc) connect() *channel.Channel {
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
		log.Println(string(msg))
		os.Exit(1)
	}
	log.Println("successfully connected to", server)
	channel := channel.New(ws, c.NewConn)
	go channel.HeartBeat(time.Second * 30)
	return channel
}

func (c *Wspc) LoadConfig(channel string) (*msg.WspConfig, error) {
	if val, ok := c.configs.Load(channel); ok {
		return val.(*msg.WspConfig), nil
	}
	return nil, fmt.Errorf(channel + " not found")
}
