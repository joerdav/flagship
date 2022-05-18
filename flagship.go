/*
A package for retreiving feature flags from a dynamo document.

Retrieving a boolean flag:

		s, err := flagship.New(context.Background(), flagship.WithTableName(tableName))
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		if s.Bool(context.Background(), "newfeature") {
			// New Code
		} else {
			// Old code
		}

*/
package flagship

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// FeatureStore defines the interface for accessing feature flags from some source.
type FeatureStore interface {
	// Bool returns the state of the feature flag with the key of `key`:
	//	if s.Bool(context.Background(), "newfeature") {
	//		// New Code
	//	} else {
	//		// Old code
	//	}
	Bool(ctx context.Context, key string) bool
}

type featureStoreConfig struct {
	TableName, RecordName, Region string
	CacheTTL                      time.Duration
	Client                        *dynamodb.Client
	Now                           func() time.Time
}

// Option is a function that can modify internal config.
type Option func(*featureStoreConfig)

// WithRegion allows modification of the AWS Region in which the dynamo table resides.
// The default value will rely on the AWS Go SDK to find the correct region.
//	s, err := flagship.New(context.Background(), flagship.WithRegion("eu-west-1"))
func WithRegion(region string) Option {
	return func(fsc *featureStoreConfig) {
		fsc.Region = region
	}
}

// WithClock allows the overriding of the function used to get current time.
// The default value will be `time.Now`.
//	s, err := flagship.New(context.Background(), flagship.WithClock(func() time.Time { return time.Time{} }))
func WithClock(clock func() time.Time) Option {
	return func(fsc *featureStoreConfig) {
		fsc.Now = clock
	}
}

// WithTableName allows modification of the AWS DynamoDB table name.
// The default value is "featureFlagStore".
//	s, err := flagship.New(context.Background(), flagship.WithTableName("feature-table"))
func WithTableName(tableName string) Option {
	return func(fsc *featureStoreConfig) {
		fsc.TableName = tableName
	}
}

// WithRecordName allows modification of the AWS DynamoDB partition key.
// The default value is "features".
//	s, err := flagship.New(context.Background(), flagship.WithRecordName("features1"))
func WithRecordName(recordName string) Option {
	return func(fsc *featureStoreConfig) {
		fsc.RecordName = recordName
	}
}

// WithTTL allows modification of the cache expiry for features.
// The default value is 30 seconds.
//	s, err := flagship.New(context.Background(), flagship.WithTTL(1 * time.Hour))
func WithTTL(ttl time.Duration) Option {
	return func(fsc *featureStoreConfig) {
		fsc.CacheTTL = ttl
	}
}

// WithClient allows modification of the dynamo client used.
// The default value is constructed using default credentials.
//	s, err := flagship.New(context.Background(), flagship.WithClient(client))
func WithClient(client *dynamodb.Client) Option {
	return func(fsc *featureStoreConfig) {
		fsc.Client = client
	}
}

// New constructs a new instance of the feature store client.
// Optionally accepts Option types as a variadic parameter:
//	s, err := flagship.New(context.Background(), flagship.WithClient(client))
func New(ctx context.Context, opts ...Option) (FeatureStore, error) {
	cfg := featureStoreConfig{
		TableName:  "featureFlagStore",
		RecordName: "features",
		CacheTTL:   time.Second * 30,
		Now:        time.Now,
	}
	for _, o := range opts {
		o(&cfg)
	}
	var dynamoOpts []func(*config.LoadOptions) error
	if cfg.Region != "" {
		dynamoOpts = append(dynamoOpts, config.WithRegion(cfg.Region))
	}
	if cfg.Client == nil {
		c, err := config.LoadDefaultConfig(context.Background(), dynamoOpts...)
		if err != nil {
			return nil, err
		}
		cfg.Client = dynamodb.NewFromConfig(c)
	}
	ds := dynamoStore{
		client:    cfg.Client,
		tableName: cfg.TableName,
		record:    cfg.RecordName,
	}
	s := featureStore{
		cacheTTL: cfg.CacheTTL,
		now:      cfg.Now,
		store:    &ds,
	}
	// Initial fetch to check it is working
	_, err := s.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("flagship - failed to fetch features: %w", err)
	}
	return &s, nil
}

type features map[string]interface{}

func (f features) Bool(s string) bool {
	b, ok := f[s].(bool)
	return b && ok
}

type store interface {
	Load(context.Context) (features, error)
}

type dynamoStore struct {
	client            *dynamodb.Client
	tableName, record string
}

func (s *dynamoStore) Load(ctx context.Context) (features, error) {
	gio, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{Value: s.record},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(gio.Item) < 1 {
		return nil, errors.New("record is empty")
	}
	var f struct {
		Features features `json:"features"`
	}
	err = unmarshalMap(gio.Item, &f)
	if err != nil {
		return nil, err
	}
	return f.Features, nil
}

type featureStore struct {
	fetchMutex     sync.Mutex
	cacheTTL       time.Duration
	expiry         time.Time
	now            func() time.Time
	cachedFeatures features
	store          store
}

func (s *featureStore) Bool(ctx context.Context, key string) bool {
	f, err := s.fetch(ctx)
	if err != nil {
		f = s.cachedFeatures
	}
	return f.Bool(key)
}

func (s *featureStore) fetch(ctx context.Context) (features, error) {
	s.fetchMutex.Lock()
	defer s.fetchMutex.Unlock()
	if s.now().Before(s.expiry) {
		return s.cachedFeatures, nil
	}
	f, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	s.expiry = s.now().Add(s.cacheTTL)
	s.cachedFeatures = f
	return s.cachedFeatures, nil
}

func unmarshalMap(m map[string]types.AttributeValue, out interface{}) error {
	return attributevalue.NewDecoder(func(do *attributevalue.DecoderOptions) { do.TagKey = "json" }).Decode(&types.AttributeValueMemberM{Value: m}, out)
}
