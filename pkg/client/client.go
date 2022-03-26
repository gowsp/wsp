// Package client for wsp
package client

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

func NewWspc(config *Config) *Wspc {
	wspc := &Wspc{Config: config, routing: proxy.NewRouting()}
	return wspc
}

type Wspc struct {
	Config  *Config
	channel sync.Map
	routing *proxy.Routing
	wan     *proxy.Wan
	closed  uint32
}

func (c *Wspc) Routing() *proxy.Routing {
	return c.routing
}
func (c *Wspc) connectWs() error {
	headers := make(http.Header)
	headers.Set("Auth", c.Config.Auth)
	headers.Set("Proto", msg.PROTOCOL_VERSION.String())
	ws, resp, err := websocket.Dial(context.Background(),
		c.Config.Server, &websocket.DialOptions{HTTPHeader: headers})
	if resp.StatusCode == 400 || resp.StatusCode == 401 {
		defer resp.Body.Close()
		msg, _ := io.ReadAll(resp.Body)
		log.Print(string(msg))
		os.Exit(1)
	}
	if err != nil {
		c.retry()
		return err
	}
	c.wan = proxy.NewWan(ws, c)
	go c.wan.HeartBeat(time.Second * 30)
	return err
}

func (c *Wspc) Interrupt() {
	atomic.AddUint32(&c.closed, 1)
	c.routing.Close()
	c.wan.Close()
}
func (c *Wspc) ListenAndServe() {
	err := c.connectWs()
	if err != nil {
		log.Println(err)
		return
	}
	c.forward()
	c.wan.Serve()
}

var startOnce sync.Once

func (c *Wspc) forward() {
	c.RemoteForward()
	startOnce.Do(func() {
		c.LocalForward()
		c.DynamicForward()
	})
}
func (c *Wspc) Close() error {
	c.retry()
	return nil
}
func (c *Wspc) retry() {
	if atomic.LoadUint32(&c.closed) > 0 {
		return
	}
	log.Println("reconnect server ...")
	time.Sleep(3 * time.Second)
	c.ListenAndServe()
}
func (c *Wspc) NewConn(message *msg.WspMessage) error {
	var reqeust msg.WspRequest
	err := proto.Unmarshal(message.Data, &reqeust)
	if err != nil {
		return err
	}
	config, err := reqeust.ToConfig()
	if err != nil {
		return err
	}
	return c.NewRemoteConn(message.Id, config)
}
