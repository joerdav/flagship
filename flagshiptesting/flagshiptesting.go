package flagshiptesting

import (
	"context"
	"io"

	"github.com/joerdav/flagship"
)

// compile time check that MockFeatureStore implements flagship.FeatureStore.
var _ flagship.FeatureStore = MockFeatureStore{}

// MockFeatureStore is used for testing feature flags. It conforms to both BoolFeatureStore and ThrottleFeatureStore.
//
//	m := MockFeatureStore{
//		"featureA":true,
//	}
//	m.Bool(context.Background(), "featureA") // true
//	m.Bool(context.Background(), "featureB") // false
//	m.ThrottleHash(context.Background(), "featureA", strings.NewReader("")) // true
//	m.ThrottleHash(context.Background(), "featureB", strings.NewReader("")) // false
type MockFeatureStore map[string]bool

func (s MockFeatureStore) Bool(_ context.Context, key string) bool {
	return s[key]
}

func (s MockFeatureStore) AllBools(_ context.Context) map[string]bool {
	return s
}

func (s MockFeatureStore) ThrottleAllow(_ context.Context, key string, _ io.Reader) bool {
	return s[key]
}
func (MockFeatureStore) GetHash(_ context.Context, _ string, _ io.Reader) uint {
	return 0
}
