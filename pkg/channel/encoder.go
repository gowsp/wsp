package channel

import (
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
)

func encode(id string, cmd msg.WspCmd, data []byte) []byte {
	msg := &msg.WspMessage{Id: id, Cmd: cmd, Data: data}
	res, err := proto.Marshal(msg)
	if err != nil {
		log.Println("error wrap message")
	}
	return res
}
