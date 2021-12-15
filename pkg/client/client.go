package client

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type Wspc struct {
	Config  *Config
	network sync.Map
	wan     *pkg.Wan
	routing *pkg.Routing
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
	c.wan = pkg.NewWan(ws)
	go c.wan.HeartBeat(time.Second * 30)
	c.routing = pkg.NewRouting()
	return err
}

func (c *Wspc) ListenAndServe() {
	err := c.connectWs()
	if err != nil {
		log.Println(err)
		return
	}
	if c.Config.Socks5 != "" {
		go c.ListenSocks5()
	}
	c.readConfig()
	c.forward()
}

func (c *Wspc) readConfig() {
	for _, val := range c.Config.Addrs {
		switch val.Forward {
		case "local":
			c.wan.SecretDail(val.Name, msg.WspType_LOCAL, val.Name, val.Secret)
			log.Printf("register local address %s", val.Name)
			c.network.Store(val.Name, val)
		case "remote":
			log.Printf("listen local address %v", val.Name)
			go c.ListenRemote(val)
		default:
			log.Println("unknown type", val.Forward)
		}
	}
}
func (c *Wspc) forward() {
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
	log.Println("reconnect server ...")
	time.Sleep(3 * time.Second)
	c.ListenAndServe()
}
func (c *Wspc) process(message *msg.Data) {
	switch message.Cmd() {
	case msg.WspCmd_CONN_REQ:
		c.NewConn(message.Msg)
	default:
		err := c.routing.Routing(message)
		if errors.Is(err, pkg.ConnNotExist) {
			c.wan.CloseRemote(message.Id(), err.Error())
		}
	}
}

func (c *Wspc) NewConn(message *msg.WspMessage) {
	var addr msg.WspAddr
	err := proto.Unmarshal(message.Data, &addr)
	if err != nil {
		log.Println("error unmarshal addr", err)
		return
	}
	switch addr.Type {
	case msg.WspType_LOCAL:
		go c.NewLocalConn(message.Id, addr.Address)
	default:
		c.wan.CloseRemote(message.Id, fmt.Sprintf("unkonwn %v", addr.Type))
	}
}
