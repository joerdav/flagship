package flagship_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/joerdav/flagship"
)

func setThrottle(d *dynamodb.Client, key string, value float64, whitelist []string, table, record string) error {
	k := &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
		"probability": &types.AttributeValueMemberN{Value: fmt.Sprint(value)},
	}}
	if len(whitelist) > 0 {
		k.Value["whitelist"] = &types.AttributeValueMemberNS{Value: whitelist}
	}
	_, err := d.PutItem(context.Background(), &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{Value: record},
			"throttles": &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					key: k,
				},
			},
		},
		TableName: &table,
	})
	return err
}

func setFlag(d *dynamodb.Client, key string, value bool, table, record string) error {
	_, err := d.PutItem(context.Background(), &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{Value: record},
			"features": &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"someflag": &types.AttributeValueMemberBOOL{Value: value},
				},
			},
		},
		TableName: &table,
	})
	return err
}

func TestCache(t *testing.T) {
	t.Parallel()
	testClient, testRegion, err := newTestClient()
	if err != nil {
		t.Fatal(err)
	}
	tableName := createLocalTable(t, testClient)
	t.Cleanup(func() {
		deleteLocalTable(t, testClient, tableName)
	})
	tests := []struct {
		name         string
		ttl          time.Duration
		expectedBool bool
	}{
		{
			name:         "given cache expires, return new value",
			ttl:          0,
			expectedBool: true,
		},
		{
			name:         "given cache does not expire, return old value",
			ttl:          time.Hour * 2,
			expectedBool: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			record := uuid.New().String()
			err := setFlag(testClient, "someflag", false, tableName, record)
			if err != nil {
				t.Errorf("unexpected error got %v", err)
			}
			currentTime := time.Time{}
			store, err := flagship.New(context.Background(),
				flagship.WithClient(testClient),
				flagship.WithTableName(tableName),
				flagship.WithRecordName(record),
				flagship.WithRegion(testRegion),
				flagship.WithTTL(tt.ttl),
				flagship.WithClock(func() time.Time {
					return currentTime
				}))
			if err != nil {
				t.Errorf("unexpected error got %v", err)
			}
			b := store.Bool(context.Background(), "someflag")
			if b {
				t.Errorf("expected flag to be false, was true")
			}
			err = setFlag(testClient, "someflag", true, tableName, record)
			if err != nil {
				t.Errorf("unexpected error got %v", err)
			}
			currentTime = currentTime.Add(time.Hour)
			if b := store.Bool(context.Background(), "someflag"); b != tt.expectedBool {
				t.Errorf("expected flag to be %v, was %v", tt.expectedBool, b)
			}
		})
	}
}

func TestAllowThrottle(t *testing.T) {
	testClient, testRegion, err := newTestClient()
	if err != nil {
		t.Fatal(err)
	}
	tableName := createLocalTable(t, testClient)
	t.Cleanup(func() {
		deleteLocalTable(t, testClient, tableName)
	})
	tests := []struct {
		name           string
		throttleKey    string
		probability    float64
		whitelist      []uint
		hashValue      string
		givenKey       string
		expectedResult bool
	}{
		{
			name:           "given throttleKey does not exist should not allow",
			throttleKey:    "someFeature",
			probability:    100,
			givenKey:       "otherFeature",
			hashValue:      "an input",
			expectedResult: false,
		},
		{
			name:           "given throttle probability is 0 always disallow",
			throttleKey:    "someFeature",
			probability:    0,
			givenKey:       "someFeature",
			hashValue:      "an input",
			expectedResult: false,
		},
		{
			name:           "given throttle probability is 100 always allow",
			throttleKey:    "someFeature",
			probability:    100,
			givenKey:       "someFeature",
			hashValue:      "an input",
			expectedResult: true,
		},
		{
			name:           "given hash is within throttle return true",
			throttleKey:    "someFeature",
			probability:    50,
			givenKey:       "someFeature",
			hashValue:      "an input",
			expectedResult: true,
		},
		{
			name:           "given hash is within whitelist return true",
			throttleKey:    "someFeature",
			probability:    0,
			whitelist:      []uint{1898},
			givenKey:       "someFeature",
			hashValue:      "an input",
			expectedResult: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			record := uuid.New().String()
			var wl []string
			for _, f := range tt.whitelist {
				wl = append(wl, fmt.Sprint(f))
			}
			err := setThrottle(testClient, tt.throttleKey, tt.probability, wl, tableName, record)
			if err != nil {
				t.Errorf("unexpected error got %v", err)
			}
			store, err := flagship.New(
				context.Background(),
				flagship.WithClient(testClient),
				flagship.WithTableName(tableName),
				flagship.WithRecordName(record),
				flagship.WithRegion(testRegion),
			)
			if err != nil {
				t.Errorf("unexpected error got %v", err)
			}
			r := store.ThrottleAllow(context.Background(), tt.givenKey, strings.NewReader(tt.hashValue))
			if r != tt.expectedResult {
				hash := store.GetHash(context.Background(), tt.givenKey, strings.NewReader(tt.hashValue))
				t.Errorf("expected flag to be %v, was %v. hash: %v", tt.expectedResult, r, hash)
			}
		})
	}
}
func TestBool(t *testing.T) {
	testClient, testRegion, err := newTestClient()
	if err != nil {
		t.Fatal(err)
	}
	tableName := createLocalTable(t, testClient)
	t.Cleanup(func() {
		deleteLocalTable(t, testClient, tableName)
	})
	tests := []struct {
		name         string
		flags        types.AttributeValue
		expectedBool bool
	}{
		{
			name:         "given empty flag, return false",
			flags:        &types.AttributeValueMemberM{Value: make(map[string]types.AttributeValue)},
			expectedBool: false,
		},
		{
			name: "given false flag, return false",
			flags: &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"someflag": &types.AttributeValueMemberBOOL{Value: false},
				},
			},
			expectedBool: false,
		},
		{
			name: "given true flag, return true",
			flags: &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"someflag": &types.AttributeValueMemberBOOL{Value: true},
				},
			},
			expectedBool: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			record := uuid.New().String()
			_, err := testClient.PutItem(context.Background(), &dynamodb.PutItemInput{
				Item: map[string]types.AttributeValue{
					"_pk":      &types.AttributeValueMemberS{Value: record},
					"features": tt.flags,
				},
				TableName: &tableName,
			})
			if err != nil {
				t.Errorf("unexpected error got %v", err)
			}
			store, err := flagship.New(
				context.Background(),
				flagship.WithClient(testClient),
				flagship.WithTableName(tableName),
				flagship.WithRecordName(record),
				flagship.WithRegion(testRegion),
			)
			if err != nil {
				t.Errorf("unexpected error got %v", err)
			}
			b := store.Bool(context.Background(), "someflag")
			if b != tt.expectedBool {
				t.Errorf("expected flag to be %v, was %v", tt.expectedBool, b)
			}
		})
	}
}

func TestNew(t *testing.T) {
	testClient, _, err := newTestClient()
	if err != nil {
		t.Fatal(err)
	}
	tableName := createLocalTable(t, testClient)
	t.Cleanup(func() {
		deleteLocalTable(t, testClient, tableName)
	})
	t.Run("if dynamo connection fails should return error", func(t *testing.T) {
		t.Parallel()
		_, err := flagship.New(context.Background(), flagship.WithClient(testClient))
		if err == nil {
			t.Errorf("expected an error got %v", err)
		}
	})
}
