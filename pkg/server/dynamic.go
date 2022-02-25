package server

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
)

func (r *Router) NewDynamic(id string, conf *msg.WspConfig) error {
	addr := conf.Address()
	log.Println("open proxy", addr)

	conn, err := net.DialTimeout(conf.Network(), addr, 5*time.Second)
	if err != nil {
		return err
	}

	in := r.wan.NewWriter(id)
	repeater := proxy.NewNetRepeater(in, conn)
	r.routing.AddRepeater(id, repeater)
	defer r.routing.Delete(id)

	if err = r.wan.Succeed(id); err != nil {
		conn.Close()
		return err
	}
	repeater.Copy()
	log.Println("close proxy", addr)
	return nil
}
