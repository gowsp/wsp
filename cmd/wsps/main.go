package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/server"
)

func main() {
	config, err := parseConfig()
	if err != nil {
		logger.Error("start wsps error: %s", err)
		return
	}
	config.Log.Init()
	wsps := server.New(config)
	addr := fmt.Sprintf(":%d", config.Port)
	srv := &http.Server{Handler: wsps, Addr: addr}
	go func() {
		log.Printf("wsps will start at %s, path %s ", addr, config.Path)
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Println(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalln("wsps forced to shutdown: %s", err)
	}
	logger.Info("wsps exiting")
}
