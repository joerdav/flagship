package config

import "github.com/spf13/pflag"

type Flags struct {
	*pflag.FlagSet
	TableName  string
	RecordName string
}

func GlobalFlags() *Flags {
	f := Flags{}
	f.FlagSet = pflag.NewFlagSet("global", pflag.ExitOnError)
	f.StringVar(&f.TableName, "tableName", "featureFlagStore", "Define which dynamodb table to point to")
	f.StringVar(&f.RecordName, "recordName", "features", "Define the partition key of the feature document")
	return &f
}
