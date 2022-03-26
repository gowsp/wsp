package cmd

import (
	"fmt"

	"github.com/gowsp/wsp/pkg/msg"
)

type Version struct {
	date    string
	version string
}

func NewVersion(date, version string) Version {
	return Version{
		date:    date,
		version: version,
	}
}

func (v *Version) PrintVersion() {
	fmt.Printf("Version: %s\n", v.version)
	fmt.Printf("Release Date: %s\n", v.date)
	fmt.Printf("Protocol Version: %s\n", msg.PROTOCOL_VERSION)
}
