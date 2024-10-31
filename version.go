package main

import "fmt"

type _version struct{}

type VersionCmd struct {
	Version _version `cmd:"" help:"Show version information and exit"`
}

var version = "dev"

// Run executes the command
func (cmd *_version) Run(globals *Globals) error {
	fmt.Println(version)
	return nil
}
