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
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/joerdav/flagship/internal/dynamostore"
	"github.com/joerdav/flagship/internal/models"
)

// BoolFeatureStore defines the interface for accessing boolean typed feature flags from some source.
type BoolFeatureStore interface {
	// Bool returns the state of the feature flag with the key of `key`:
	// If the feature is missing from the table then always returns false.
	// Example:
	// {
	//     "features": {
	//         "newFeature": true
	//     }
	// }
	//	if s.Bool(context.Background(), "newfeature") {
	//		// New Code
	//	} else {
	//		// Old code
	//	}
	Bool(ctx context.Context, key string) bool
	// All returns the state containing all feature flags
	AllBools(ctx context.Context) map[string]bool
}

// ThrottleFeatureStore defines the interface for accessing a feature flag that needs bucketing.
type ThrottleFeatureStore interface {
	// ThrottleAllow returns whether a given hash key is bucketed.
	// If the feature is missing from the table then always returns false.
	// Example:
	// {
	//     "throttles": {
	//         "newThrottleFeature": {
	//             // whitelist is an optional list of hashes that will always be bucketed.
	//             "whitelist":[10, 3321],
	//             // probability is the likelihood that a hash is bucketed as a percentage.
	//             // value is truncated to 2dp
	//             "probability": 2.5
	//         }
	//     }
	// }
	//	if s.ThrottleAllow(context.Background(), "newThrottleFeature", strings.NewReader("some hash")) {
	//		// New Code
	//	} else {
	//		// Old code
	//	}
	ThrottleAllow(ctx context.Context, key string, hashKey io.Reader) bool
	// GetHash returns the hash that would be bucketed in ThrottleAllow:
	//	h := s.GetHash(context.Background(), "newThrottleFeature", strings.NewReader("some hash")) {
	GetHash(ctx context.Context, key string, hashKey io.Reader) uint
}

// FeatureStore is an aggregate interface for accessing all supported types of feature flag.
type FeatureStore interface {
	BoolFeatureStore
	ThrottleFeatureStore
}

type featureStoreConfig struct {
	TableName, RecordName, Region string
	CacheTTL                      time.Duration
	Client                        *dynamodb.Client
	Now                           func() time.Time
}

// New constructs a new instance of the feature store client.
// Optionally accepts Option types as a variadic parameter:
//
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
	ds := dynamostore.NewDynamoStoreWithClient(cfg.TableName, cfg.RecordName, cfg.Client)
	s := featureStore{
		cacheTTL: cfg.CacheTTL,
		now:      cfg.Now,
		store:    &ds,
	}
	// Initial fetch to check it is working
	_, _, err := s.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("flagship - failed to fetch features: %w", err)
	}
	return &s, nil
}

type throttleConfigInt struct {
	models.ThrottleConfig
	// Threshold is an integer representation of Probability. Floor(Probability*100)
	Threshold uint
}

type featureStore struct {
	fetchMutex      sync.Mutex
	cacheTTL        time.Duration
	expiry          time.Time
	now             func() time.Time
	cachedFeatures  models.Features
	cachedThrottles map[string]*throttleConfigInt
	store           store
}

func (s *featureStore) ThrottleAllow(ctx context.Context, key string, hashKey io.Reader) bool {
	_, ts, err := s.fetch(ctx)
	if err != nil {
		return false
	}
	t := ts[key]
	if t.Disabled {
		return false
	}
	h := s.GetHash(ctx, key, hashKey)
	if t == nil {
		return false
	}
	for _, wl := range t.Whitelist {
		if h == wl {
			return true
		}
	}
	if t.Threshold == 0 {
		return false
	}
	if t.Threshold > 100_00 {
		return true
	}
	return h <= t.Threshold

}
func GetHash(ctx context.Context, key string, hashKey io.Reader) uint {
	f := fnv.New32a()
	f.Write([]byte(key))
	_, _ = io.Copy(f, hashKey)
	return uint(f.Sum32()) % 100_00
}
func (s *featureStore) GetHash(ctx context.Context, key string, hashKey io.Reader) uint {
	return GetHash(ctx, key, hashKey)
}

func (s *featureStore) Bool(ctx context.Context, key string) bool {
	f, _, err := s.fetch(ctx)
	if err != nil {
		f = s.cachedFeatures
	}
	return f.Bool(key)
}

func (s *featureStore) AllBools(ctx context.Context) (allBools map[string]bool) {
	f, _, err := s.fetch(ctx)
	if err != nil {
		f = s.cachedFeatures
	}

	allBools = make(map[string]bool)

	for key, value := range f {
		boolValue, ok := value.(bool)
		if ok {
			allBools[key] = boolValue
		}
	}

	return
}

func (s *featureStore) fetch(ctx context.Context) (models.Features, map[string]*throttleConfigInt, error) {
	s.fetchMutex.Lock()
	defer s.fetchMutex.Unlock()
	if s.now().Before(s.expiry) {
		return s.cachedFeatures, s.cachedThrottles, nil
	}
	f, t, err := s.store.Load(ctx)
	if err != nil {
		return nil, nil, err
	}
	s.expiry = s.now().Add(s.cacheTTL)
	s.cachedFeatures = f
	s.cachedThrottles = make(map[string]*throttleConfigInt)
	for k, th := range t {
		s.cachedThrottles[k] = &throttleConfigInt{
			ThrottleConfig: th,
			Threshold:      uint(math.Floor(th.Probability * 100)),
		}
	}
	return s.cachedFeatures, s.cachedThrottles, nil
}

type store interface {
	Load(context.Context) (models.Features, map[string]models.ThrottleConfig, error)
}
