package server

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
)

func (r *Router) NewDynamic(id string, conf *msg.WspConfig) error {
	addr := conf.Address()
	log.Println("open proxy", addr)

	conn, err := net.DialTimeout(conf.Network(), addr, 5*time.Second)
	if err != nil {
		return err
	}

	_, repeater := r.wan.NewTCPChannel(id, conn)
	
	if err = r.wan.Succeed(id); err != nil {
		repeater.Interrupt()
		return err
	}
	repeater.Copy()
	log.Println("close proxy", addr)
	return nil
}
