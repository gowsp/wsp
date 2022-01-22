package server

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
)

func (r *Router) NewDynamic(id string, conf *msg.WspConfig) {
	addr := conf.Address()
	log.Println("open proxy", addr)

	conn, err := net.DialTimeout(conf.Network(), addr, 5*time.Second)
	if err != nil {
		r.wan.ReplyMessage(id, false, err.Error())
		return
	}

	in := r.wan.NewWriter(id)
	repeater := proxy.NewNetRepeater(in, conn)
	r.routing.AddRepeater(id, repeater)
	defer r.routing.Delete(id)

	if err = r.wan.Reply(id, true); err != nil {
		log.Println(err)
		conn.Close()
		return
	}
	repeater.Copy()
	log.Println("close proxy", addr)
}
