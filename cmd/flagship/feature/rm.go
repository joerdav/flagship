package feature

import (
	"context"
	"errors"
	"fmt"

	"github.com/joerdav/flagship/internal/dynamostore"
)

type Rm struct {
	Store dynamostore.DynamoStore
}

func (r Rm) Run(args []string) error {
	if len(args) < 1 {
		r.Help()
		return errors.New("No featureName provided.")
	}
	err := r.Store.RemoveFeature(context.Background(), args[0])
	if err != nil {
		return fmt.Errorf("Error when setting flag: %s", err.Error())
	}
	fmt.Printf("%v removed!\n", args[0])
	return nil
}
func (Rm) Help() {
	fmt.Println(`usage: flagship feature rm [featureName]
	Removes a feature flag.`)
}
