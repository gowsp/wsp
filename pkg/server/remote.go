package server

import (
	"fmt"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
)

func (local *Router) NewLocal(id string, conf *msg.WspConfig) error {
	key := conf.Channel()
	if val, ok := local.wsps.LoadRouter(key); ok {
		log.Printf("start bridge channel %s\n", key)
		remote := val.(*Router)

		_, ok := remote.channel.Load(key)
		if !ok {
			return fmt.Errorf("channel not registered")
		}
		remote.routing.AddPending(id, &proxy.Pending{OnReponse: func(wm *msg.Data, res *msg.WspResponse) {
			if res.Code == msg.WspCode_SUCCESS {
				remoteWirter := proxy.NewWsRepeater(id, remote.wan)
				local.routing.AddRepeater(id, remoteWirter)
				localWriter := proxy.NewWsRepeater(id, local.wan)
				remote.routing.AddRepeater(id, localWriter)
			} else {
				remote.routing.DeleteConn(id)
			}
			local.wan.Write(*wm.Raw)
		}})

		if err := remote.wan.Dail(id, conf); err != nil {
			remote.routing.DeleteConn(id)
			return err
		}
		return nil
	}
	return fmt.Errorf("channel not registered")
}
