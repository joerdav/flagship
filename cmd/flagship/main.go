package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/joerdav/flagship/cmd/flagship/config"
	"github.com/joerdav/flagship/cmd/flagship/feature"
	"github.com/joerdav/flagship/cmd/flagship/hashcmd"
	"github.com/joerdav/flagship/cmd/flagship/lscmd"
	"github.com/joerdav/flagship/internal/dynamostore"
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

type parentCommand struct {
	cmds map[string]command
	name string
}

func (pc parentCommand) Run(args []string) error {
	if len(args) < 1 {
		pc.Help()
		return nil
	}
	cmd := pc.cmds[args[0]]
	if cmd == nil {
		pc.Help()
		return errors.New("subcommand not found")
	}
	if len(args) == 1 {
		cmd.Run([]string{})
	}
	return cmd.Run(args[1:])
}

func (pc parentCommand) Help() {
	cmds := []string{}
	for c := range pc.cmds {
		cmds = append(cmds, c)
	}
	fmt.Printf(`usage: flagship %s [subcommand]
Commands: %s
`, pc.name, strings.Join(cmds, ", "))
}
func newParentCommand(name string, cmds map[string]command) parentCommand {
	return parentCommand{cmds, name}
}

func run() error {
	f := config.GlobalFlags()
	f.Parse(os.Args[1:])
	region := os.Getenv("AWS_REGION")
	store, err := dynamostore.NewDynamoStore(f.TableName, f.RecordName, region)
	if err != nil {
		return fmt.Errorf("Error when creating DynamoDB connection: %s", err.Error())
	}
	cmds := map[string]command{
		"ls":   lscmd.Command{},
		"hash": hashcmd.Command{},
		"feature": newParentCommand("sub", map[string]command{
			"get":     feature.Get{Store: store, Out: os.Stdout},
			"enable":  feature.Enable{Store: store},
			"disable": feature.Disable{Store: store},
			"rm":      feature.Rm{Store: store},
		}),
	}
	cmdl := []string{}
	for k := range cmds {
		cmdl = append(cmdl, k)
	}
	if len(os.Args) < 2 {
		usage(cmdl)
		return nil
	}
	if os.Args[1] == "version" {
		fmt.Println(goInstallVersion())
		return nil
	}
	if os.Args[1] == "help" && len(os.Args) < 3 {
		usage(cmdl)
		return nil
	}
	if os.Args[1] == "help" && len(os.Args) < 4 {
		cmd := cmds[os.Args[2]]
		if cmd == nil {
			usage(cmdl)
			return errors.New("command does not found")
		}
		cmd.Help()

		return nil
	}
	cmd := cmds[os.Args[1]]
	if cmd == nil {
		usage(cmdl)
		return errors.New("command not found")
	}
	return cmd.Run(os.Args[2:])
}

func usage(cmds []string) {
	fmt.Println(`usage: flagship <command> [parameters]
	To see help text use:
		flagship help <command>
	Commands:`, strings.Join(cmds, ", "))
}
