package feature

import (
	"context"
	"errors"
	"fmt"

	"github.com/joerdav/flagship/internal/dynamostore"
)

type Disable struct {
	Store dynamostore.DynamoStore
}

func (d Disable) Run(args []string) error {
	if len(args) < 1 {
		d.Help()
		return errors.New("No featureName provided.")
	}
	err := d.Store.SetFeature(context.Background(), args[0], false)
	if err != nil {
		return fmt.Errorf("Error when setting flag: %s", err.Error())
	}
	fmt.Printf("%v: %v\n", args[0], false)
	return nil
}
func (Disable) Help() {
	fmt.Println(`usage: flagship feature disable [featureName]
	Disables a feature flag.`)
}
