package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/gowsp/wsp/pkg/client"
	"github.com/gowsp/wsp/pkg/logger"
)

func main() {
	config, err := parseConfig()
	if err != nil {
		logger.Error("start wspc error: %s", err)
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	config.Log.Init()
	for _, wspc := range config.Client {
		go client.New(wspc).ListenAndServe()
	}
	<-ctx.Done()
	logger.Info("wspc closed")
}
