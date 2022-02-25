package proxy

import (
	"errors"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

func NewHandler(proxy Proxy) *Handler {
	return &Handler{proxy: proxy, routing: proxy.Routing(), wan: proxy.Wan()}
}

type Handler struct {
	wan     *Wan
	proxy   Proxy
	routing *Routing
}

func (h *Handler) ServeConn() {
	for {
		_, data, err := h.wan.Read()
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			log.Println("error reading webSocket message:", err)
			break
		}
		var m msg.WspMessage
		if proto.Unmarshal(data, &m) != nil {
			log.Println("error unmarshal message:", err)
			continue
		}
		h.process(&msg.Data{Msg: &m, Raw: &data})
	}
	h.proxy.Close()
}

func (h *Handler) process(data *msg.Data) {
	switch data.Cmd() {
	case msg.WspCmd_CONNECT:
		go func() {
			err := h.proxy.NewConn(data.Msg)
			if err != nil {
				h.wan.Reply(data.ID(), false, err.Error())
			}
		}()
	default:
		err := h.routing.Routing(data)
		if errors.Is(err, ErrConnNotExist) {
			h.wan.CloseRemote(data.ID(), err.Error())
		}
	}
}
