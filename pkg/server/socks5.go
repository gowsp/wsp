package server

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg"
)

func (p *Router) NewSocks5Conn(id, addr string) {
	log.Println("open connect", addr)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		p.wan.Reply(id, false)
		return
	}
	in := p.wan.NewWriter(id)

	repeater := pkg.NewNetRepeater(in, conn)
	p.routing.AddRepeater(id, repeater)
	defer p.routing.Delete(id)

	if err = p.wan.Reply(id, true); err != nil {
		log.Println(err)
		return
	}
	repeater.Copy()
	log.Println("close connect", addr)
}
