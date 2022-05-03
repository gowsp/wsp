package client

import (
	"log"
	"net"

	"github.com/gowsp/wsp/pkg/msg"
)

type DynamicProxy interface {
	// Listen local
	Listen()

	// ServeConn conn by wspc
	ServeConn(conn net.Conn) error
}

func (c *Wspc) DynamicForward() {
	for _, val := range c.config.Dynamic {
		conf, err := msg.NewWspConfig(msg.WspType_DYNAMIC, val)
		if err != nil {
			log.Println("forward dynamic error,", err)
			continue
		}
		c.ListenDynamic(conf)
	}
}

func (c *Wspc) ListenDynamic(conf *msg.WspConfig) {
	var proxy DynamicProxy
	switch conf.Scheme() {
	case "http":
		proxy = &HTTPProxy{conf: conf, wspc: c}
	case "socks5":
		proxy = &Socks5Proxy{conf: conf, wspc: c}
	default:
		log.Println("Not supported", conf.Scheme())
		return
	}
	go proxy.Listen()
}
