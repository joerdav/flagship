package flagship_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/joerdav/flagship"
)

func setFlag(key string, value bool, table, record string) error {
	_, err := testClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{Value: record},
			"features": &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"someflag": &types.AttributeValueMemberBOOL{Value: value},
			}},
		},
		TableName: &table,
	})
	return err
}

func TestNew(t *testing.T) {
	tableName := createLocalTable(t)
	t.Cleanup(func() {
		deleteLocalTable(t, tableName)
	})
	t.Run("if dynamo connection fails should return error", func(t *testing.T) {
		_, err := flagship.New(context.Background(), flagship.WithClient(testClient))
		if err == nil {
			t.Errorf("expected an error got %v", err)
		}
	})
	t.Run("if loaded successfully, but blank flag then return false", func(t *testing.T) {
		r := uuid.New().String()
		_, err := testClient.PutItem(context.Background(), &dynamodb.PutItemInput{
			Item: map[string]types.AttributeValue{
				"_pk":      &types.AttributeValueMemberS{Value: r},
				"features": &types.AttributeValueMemberM{Value: make(map[string]types.AttributeValue)},
			},
			TableName: &tableName,
		})
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		s, err := flagship.New(context.Background(), flagship.WithClient(testClient), flagship.WithTableName(tableName), flagship.WithRecordName(r), flagship.WithRegion(testRegion))
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		b := s.Bool(context.Background(), "someflag")
		if b {
			t.Errorf("expected flag to be false, was true")
		}
	})
	t.Run("if loaded successfully, and flag is true return true", func(t *testing.T) {
		r := uuid.New().String()
		err := setFlag("someflag", true, tableName, r)
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		s, err := flagship.New(context.Background(), flagship.WithClient(testClient), flagship.WithTableName(tableName), flagship.WithRecordName(r), flagship.WithRegion(testRegion))
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		b := s.Bool(context.Background(), "someflag")
		if !b {
			t.Errorf("expected flag to be true, was false")
		}
	})
	t.Run("if loaded successfully, and flag is false return false", func(t *testing.T) {
		r := uuid.New().String()
		err := setFlag("someflag", false, tableName, r)
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		s, err := flagship.New(context.Background(), flagship.WithClient(testClient), flagship.WithTableName(tableName), flagship.WithRecordName(r), flagship.WithRegion(testRegion))
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		b := s.Bool(context.Background(), "someflag")
		if b {
			t.Errorf("expected flag to be false, was true")
		}
	})
	t.Run("if response is cached and not expired then return cached value", func(t *testing.T) {
		r := uuid.New().String()
		err := setFlag("someflag", false, tableName, r)
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		ct := time.Time{}
		s, err := flagship.New(context.Background(),
			flagship.WithClient(testClient),
			flagship.WithTableName(tableName),
			flagship.WithRecordName(r),
			flagship.WithRegion(testRegion),
			flagship.WithTTL(time.Hour),
			flagship.WithClock(func() time.Time {
				return ct
			}))
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		b := s.Bool(context.Background(), "someflag")
		if b {
			t.Errorf("expected flag to be false, was true")
		}
		err = setFlag("someflag", true, tableName, r)
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		if s.Bool(context.Background(), "someflag") {
			t.Errorf("expected flag to be false, was true")
		}
	})
	t.Run("if response is cached and expired then return new value", func(t *testing.T) {
		r := uuid.New().String()
		err := setFlag("someflag", false, tableName, r)
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		ct := time.Time{}
		s, err := flagship.New(context.Background(),
			flagship.WithClient(testClient),
			flagship.WithTableName(tableName),
			flagship.WithRecordName(r),
			flagship.WithRegion(testRegion),
			flagship.WithTTL(time.Hour),
			flagship.WithClock(func() time.Time {
				return ct
			}))
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		b := s.Bool(context.Background(), "someflag")
		if b {
			t.Errorf("expected flag to be false, was true")
		}
		err = setFlag("someflag", true, tableName, r)
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		ct = ct.Add(2 * time.Hour)
		if !s.Bool(context.Background(), "someflag") {
			t.Errorf("expected flag to be true, was false")
		}
	})
}
