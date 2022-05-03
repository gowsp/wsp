package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/gowsp/wsp/internal/cmd"
	"github.com/gowsp/wsp/pkg/client"
)

var date string
var version string

func main() {
	config := &client.Config{}
	err := cmd.ParseConfig(config, cmd.NewVersion(date, version))
	if err != nil {
		log.Println(err)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	client := client.New(config)
	go client.ListenAndServe()
	<-ctx.Done()
	log.Println("wspc closed")
}
