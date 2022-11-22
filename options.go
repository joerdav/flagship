package flagship

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Option is a function that can modify internal config.
type Option func(*featureStoreConfig)

// WithRegion allows modification of the AWS Region in which the dynamo table resides.
// The default value will rely on the AWS Go SDK to find the correct region.
//
//	s, err := flagship.New(context.Background(), flagship.WithRegion("eu-west-1"))
func WithRegion(region string) Option {
	return func(fsc *featureStoreConfig) {
		fsc.Region = region
	}
}

// WithClock allows the overriding of the function used to get current time.
// The default value will be `time.Now`.
//
//	s, err := flagship.New(context.Background(), flagship.WithClock(func() time.Time { return time.Time{} }))
func WithClock(clock func() time.Time) Option {
	return func(fsc *featureStoreConfig) {
		fsc.Now = clock
	}
}

// WithTableName allows modification of the AWS DynamoDB table name.
// The default value is "featureFlagStore".
//
//	s, err := flagship.New(context.Background(), flagship.WithTableName("feature-table"))
func WithTableName(tableName string) Option {
	return func(fsc *featureStoreConfig) {
		fsc.TableName = tableName
	}
}

// WithRecordName allows modification of the AWS DynamoDB partition key.
// The default value is "features".
//
//	s, err := flagship.New(context.Background(), flagship.WithRecordName("features1"))
func WithRecordName(recordName string) Option {
	return func(fsc *featureStoreConfig) {
		fsc.RecordName = recordName
	}
}

// WithTTL allows modification of the cache expiry for features.
// The default value is 30 seconds.
//
//	s, err := flagship.New(context.Background(), flagship.WithTTL(1 * time.Hour))
func WithTTL(ttl time.Duration) Option {
	return func(fsc *featureStoreConfig) {
		fsc.CacheTTL = ttl
	}
}

// WithClient allows modification of the dynamo client used.
// The default value is constructed using default credentials.
//
//	s, err := flagship.New(context.Background(), flagship.WithClient(client))
func WithClient(client *dynamodb.Client) Option {
	return func(fsc *featureStoreConfig) {
		fsc.Client = client
	}
}

// WithLogger allows the logging of flagship internals
// The default value is nil
//
//	s, err := flagship.New(context.Background(), flagship.WithLogger(logger))
func WithLogger(logger *log.Logger) Option {
	return func(fsc *featureStoreConfig) {
		fsc.Logger = logger
	}
}
