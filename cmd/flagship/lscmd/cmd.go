package lscmd

import "fmt"

type Command struct{}

func (Command) Run(args []string) error {
	fmt.Println("ls")
	return nil
}

func (Command) Help() {
	fmt.Println(`usage: flagship ls
	Returns the status of all feature flags.`)
}
