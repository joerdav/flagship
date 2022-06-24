package hashcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/joerdav/flagship"
)

type Command struct{}

func (Command) Help() {
	fmt.Println(`
	usage: flagship hash <throttle> <hash input>
	Returns the calculated hash value given an input and a throttle.
	Useful for constructing whitelists.
	`[1:])
}
func (c Command) Run(args []string) error {
	if len(args) != 2 {
		c.Help()
		return nil
	}
	fmt.Printf("Calculated Hash: %v", flagship.GetHash(context.Background(), args[0], strings.NewReader(args[1])))
	fmt.Println()
	return nil
}
