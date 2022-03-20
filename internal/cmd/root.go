package cmd

import (
	"encoding/json"
	"flag"
	"os"
)

var (
	configFile   string
	printVersion bool
)

type CmdConfig interface {
	BindFlag()
	NewEmpty() CmdConfig
	Merge(c CmdConfig)
}

func ParseConfig(config CmdConfig, version Version) error {
	config.BindFlag()
	flag.BoolVar(&printVersion, "v", false, "print version and exit")
	flag.StringVar(&configFile, "F", "wsp.json", "Specifies an alternative per-user configuration file")
	flag.Parse()

	if printVersion {
		version.PrintVersion()
		os.Exit(0)
	}
	if configFile == "" {
		return nil
	}
	conf, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	temp := config.NewEmpty()
	err = json.Unmarshal(conf, temp)
	if err != nil {
		return err
	}
	config.Merge(temp)
	return nil
}
