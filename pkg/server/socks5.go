package server

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg"
)

func (p *Router) NewSocks5Conn(id, addr string) {
	log.Println("open connect", addr)
	out, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		p.wan.Reply(id, false)
		return
	}
	in := p.wan.NewWriter(id)

	bridge := pkg.NewLanBridge(in, out)
	p.lan.Store(id, bridge)
	defer p.lan.Delete(id)

	if err = p.wan.Reply(id, true); err != nil {
		log.Println(err)
		return
	}
	bridge.Forward()
	log.Println("close connect", addr)
}
