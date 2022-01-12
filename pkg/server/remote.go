package server

import (
	"fmt"
	"log"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
)

func (r *Router) NewRemoteConn(id string, addr *msg.WspAddr) {
	if val, ok := r.GlobalHub().LoadLocal(addr.Address); ok {
		log.Printf("start bridge on %s\n", addr.Address)
		l := val.(*Router)

		registered, ok := l.hub.LoadLocal(addr.Address)
		if !ok {
			r.GlobalHub().local.Delete(addr.Address)
			r.wan.CloseRemote(id, fmt.Sprintf("net %s not registered", addr.Address))
			return
		}
		if registered.(*msg.WspAddr).Secret != addr.Secret {
			r.wan.CloseRemote(id, fmt.Sprintf("net %s secret do not match", addr.Address))
			return
		}

		inBrige := pkg.NewWsRepeater(id, l.wan)
		r.routing.AddRepeater(id, inBrige)

		outBrige := pkg.NewWsRepeater(id, r.wan)
		l.routing.AddRepeater(id, outBrige)
		l.routing.AddPending(id, &pkg.Pending{OnReponse: func(wm *msg.Data) {
			outBrige.Relay(wm)
		}})

		err := l.wan.Dail(id, msg.WspType_LOCAL, addr.Address)
		if err != nil {
			l.routing.Delete(id)
			r.routing.Delete(id)
			r.wan.CloseRemote(id, fmt.Sprintf("name %s connect error", addr.Address))
			return
		}
	} else {
		r.wan.CloseRemote(id, fmt.Sprintf("name %s not register", addr.Address))
	}
}
