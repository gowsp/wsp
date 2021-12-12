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
	conn, err := net.DialTimeout("tcp", addr.Address(), 5*time.Second)
	if err != nil {
		c.wan.CloseRemote(id, fmt.Sprintf("error connect dial %s", address))
		return
	}

	in := c.wan.NewWriter(id)
	repeater := pkg.NewNetRepeater(in, conn)
	c.routing.AddRepeater(id, repeater)
	defer c.routing.Delete(id)

	c.wan.Reply(id, true)

	repeater.Copy()
	log.Println("close local connect", address)
}
