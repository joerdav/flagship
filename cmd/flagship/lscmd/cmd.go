package lscmd

import (
	"context"
	"fmt"
	"os"

	"github.com/joerdav/flagship/cmd/flagship/config"
	"github.com/joerdav/flagship/internal/dynamostore"
)

type Command struct{}

func (Command) Run(args []string) error {
	f := config.GlobalFlags()
	f.Parse(args)
	region := os.Getenv("AWS_REGION")
	store, err := dynamostore.NewDynamoStore(f.TableName, f.RecordName, region)
	if err != nil {
		return fmt.Errorf("Error when creating DynamoDB connection: %s", err.Error())
	}
	features, throttles, err := store.Load(context.Background())
	if err != nil {
		return fmt.Errorf("Error when loading document: %s", err.Error())
	}
	fmt.Println("Features:")
	for f, v := range features {
		b, ok := v.(bool)
		if !ok {
			fmt.Printf("	%s: (not a boolean)]\n", f)
			continue
		}
		fmt.Printf("	%s: %v\n", f, b)
	}
	fmt.Println("Throttles:")
	for f, v := range throttles {
		fmt.Printf("	%s:\n", f)
		fmt.Printf("		Probability: %v\n", v.Probability)
		fmt.Print("		Whitelist: [ ")
		for i, w := range v.Whitelist {
			fmt.Print(w)
			if i != len(v.Whitelist)-1 {
				fmt.Print(", ")
			}
		}
		fmt.Print(" ]")
		fmt.Println()
	}
	return nil
}

func (Command) Help() {
	fmt.Println(`usage: flagship ls
	Returns the status of all feature flags.`)
}
