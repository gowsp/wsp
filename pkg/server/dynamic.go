package server

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
)

func (p *Router) NewDynamic(id string, conf *msg.WspConfig) {
	addr := conf.Address()
	log.Println("open proxy", addr)

	conn, err := net.DialTimeout(conf.Network(), addr, 5*time.Second)
	if err != nil {
		p.wan.ReplyMessage(id, false, err.Error())
		return
	}

	in := p.wan.NewWriter(id)
	repeater := proxy.NewNetRepeater(in, conn)
	p.routing.AddRepeater(id, repeater)
	defer p.routing.Delete(id)

	if err = p.wan.Reply(id, true); err != nil {
		log.Println(err)
		return
	}
	repeater.Copy()
	log.Println("close proxy", addr)
}
