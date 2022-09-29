package feature

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/joerdav/flagship/internal/dynamostore"
)

type Get struct {
	Store dynamostore.DynamoStore
	Out   io.Writer
}

func (g Get) Run(args []string) error {
	if len(args) < 1 {
		g.Help()
		return errors.New("No featureName provided.")
	}
	features, _, err := g.Store.Load(context.Background())
	if err != nil {
		return fmt.Errorf("Error loading features: %s", err.Error())
	}
	fe, ok := features[args[0]]
	if !ok {
		return fmt.Errorf("No feature found: %s", args[0])
	}
	fmt.Fprintf(g.Out, "%s: %v\n", args[0], fe)
	return nil
}
func (g Get) Help() {
	fmt.Fprintln(g.Out, `usage: flagship feature get [featureName]
	Returns the status of all feature flags.
	Optionally tableName and recordName can be provided. (default=featureFlagStore, features)`)
}
