package client

import (
	"log"
	"net"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
)

func (c *Wspc) ListenRemote(addr Addr) {
	local, err := net.Listen("tcp", addr.Address())
	if err != nil {
		log.Println(err)
		return
	}
	for {
		conn, err := local.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		c.NewRemoteConn(conn, addr.Name, addr.Secret)
	}
}

func (c *Wspc) NewRemoteConn(conn net.Conn, address, secret string) {
	log.Println("open remote connect", address)
	id := ksuid.New().String()

	in := c.wan.NewWriter(id)
	repeater := pkg.NewNetRepeater(in, conn)
	c.routing.AddRepeater(id, repeater)

	c.routing.AddPending(id, &pkg.Pending{OnReponse: func(message *msg.Data) {
		defer c.routing.Delete(id)
		defer log.Println("close remote connect", address)
		if message.Payload()[0] == 0 {
			repeater.Interrupt()
			return
		}
		repeater.Copy()
	}})
	
	err := c.wan.SecretDail(id, msg.WspType_REMOTE, address, secret)
	if err != nil {
		repeater.Interrupt()
		c.routing.Delete(id)
		log.Println(err)
		return
	}
}
