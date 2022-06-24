package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/joerdav/flagship/cmd/flagship/hashcmd"
	"github.com/joerdav/flagship/cmd/flagship/lscmd"
)

// Source builds use this value. When installed using `go install github.com/joerdav/flagship/cmd/flagship@latest` the `version` variable is empty, but
// the debug.ReadBuildInfo return value provides the package version number installed by `go install`
func goInstallVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	return info.Main.Version
}

func main() {
	if err := run(); err != nil {
		fmt.Printf("An error occured: %v\n", err)
		os.Exit(1)
	}
}

type command interface {
	Run(args []string) error
	Help()
}

func run() error {
	cmds := map[string]command{
		"ls":   lscmd.Command{},
		"hash": hashcmd.Command{},
	}
	if len(os.Args) < 2 {
		usage()

		return nil
	}
	if os.Args[1] == "version" {
		fmt.Println(goInstallVersion())
		return nil
	}
	if os.Args[1] == "help" && len(os.Args) < 3 {
		usage()
		return nil
	}
	if os.Args[1] == "help" && len(os.Args) < 4 {
		cmd := cmds[os.Args[2]]
		if cmd == nil {
			usage()
			return errors.New("command does not found")
		}
		cmd.Help()

		return nil
	}
	cmd := cmds[os.Args[1]]
	if cmd == nil {
		usage()

		return errors.New("command does not found")
	}
	return cmd.Run(os.Args[2:])
}

func usage() {
	fmt.Println(`usage: flagship <command> [parameters]
	To see help text use:
		flagship help <command>
	Commands:
		ls
		get
		set`)
}
