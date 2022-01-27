// Package client for wsp
package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type Config struct {
	Auth    string   `json:"auth,omitempty"`
	Server  string   `json:"server,omitempty"`
	Local   []string `json:"local,omitempty"`
	Remote  []string `json:"remote,omitempty"`
	Dynamic []string `json:"dynamic,omitempty"`
}

type Wspc struct {
	Config  *Config
	channel sync.Map
	routing *proxy.Routing
	wan     *proxy.Wan
	closed  uint32
}

func (c *Wspc) connectWs() error {
	headers := make(http.Header)
	headers.Set("Auth", c.Config.Auth)
	ws, resp, err := websocket.Dial(context.Background(),
		c.Config.Server, &websocket.DialOptions{HTTPHeader: headers})
	if err != nil {
		c.retry()
		return err
	}
	if resp.Status == "403" {
		return fmt.Errorf("error auth")
	}
	c.wan = proxy.NewWan(ws)
	go c.wan.HeartBeat(time.Second * 30)
	c.routing = proxy.NewRouting()
	return err
}

func (c *Wspc) Close() {
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
	c.start()
}

func (c *Wspc) forward() {
	c.LocalForward()
	c.RemoteForward()
	c.DynamicForward()
}
func (c *Wspc) start() {
	for {
		_, data, err := c.wan.Read()
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			log.Println("error reading webSocket message:", err)
			break
		}
		var m msg.WspMessage
		err = proto.Unmarshal(data, &m)
		if err != nil {
			log.Println("error unmarshal message:", err)
			continue
		}
		c.process(&msg.Data{Msg: &m, Raw: &data})
	}
	c.retry()
}
func (c *Wspc) retry() {
	if atomic.LoadUint32(&c.closed) > 0 {
		return
	}
	log.Println("reconnect server ...")
	time.Sleep(3 * time.Second)
	c.ListenAndServe()
}
func (c *Wspc) process(message *msg.Data) {
	switch message.Cmd() {
	case msg.WspCmd_CONNECT:
		go func() {
			err := c.NewConn(message.Msg)
			if err != nil {
				c.wan.ReplyMessage(message.ID(), false, err.Error())
			}
		}()
	default:
		err := c.routing.Routing(message)
		if errors.Is(err, proxy.ErrConnNotExist) {
			c.wan.CloseRemote(message.ID(), err.Error())
		}
	}
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
