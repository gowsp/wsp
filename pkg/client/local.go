package client

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg"
)

func (c *Wspc) NewLocalConn(id, address string) {
	log.Println("local recive connect", address)
	val, ok := c.network.Load(address)
	if !ok {
		c.wan.CloseRemote(id, address+" not found")
		return
	}
	addr := val.(Addr)
	out, err := net.DialTimeout("tcp", addr.Address(), 5*time.Second)
	if err != nil {
		c.wan.CloseRemote(id, fmt.Sprintf("error connect dial %s", address))
		return
	}

	in := c.wan.NewWriter(id)
	bridge := pkg.NewLanBridge(in, out)
	c.lan.Store(id, bridge)
	defer c.lan.Delete(id)
	
	c.wan.Reply(id, true)

	bridge.Forward()
	log.Println("close local connect", address)
}
