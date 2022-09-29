package feature

import (
	"context"
	"errors"
	"fmt"

	"github.com/joerdav/flagship/internal/dynamostore"
)

type Enable struct {
	Store dynamostore.DynamoStore
}

func (e Enable) Run(args []string) error {
	if len(args) < 1 {
		e.Help()
		return errors.New("No featureName provided.")
	}
	err := e.Store.SetFeature(context.Background(), args[0], true)
	if err != nil {
		return fmt.Errorf("Error when setting flag: %s", err.Error())
	}
	fmt.Printf("%v: %v\n", args[0], true)
	return nil
}
func (Enable) Help() {
	fmt.Println(`usage: flagship feature enable [featureName]
	Enables a feature flag.`)
}
