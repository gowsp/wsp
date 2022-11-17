package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/server"
)

var date string
var version string

var (
	configFile  string
	showVersion bool
)

func parseConfig() (*server.Config, error) {
	flag.BoolVar(&showVersion, "v", false, "print version and exit")
	flag.StringVar(&configFile, "c", "wsps.json", "Specifies an alternative per-user configuration file")
	flag.Parse()

	if showVersion {
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Release Date: %s\n", date)
		fmt.Printf("Protocol Version: %s\n", msg.PROTOCOL_VERSION)
		os.Exit(0)
	}
	if configFile == "" {
		return nil, errors.New("config file does not exist")
	}
	conf, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	config := new(server.Config)
	err = json.Unmarshal(conf, config)
	return config, err
}
