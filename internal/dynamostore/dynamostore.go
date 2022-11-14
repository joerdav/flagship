package dynamostore

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/joerdav/flagship/internal/models"
)

type DynamoStore struct {
	Client            *dynamodb.Client
	TableName, Record string
}

func NewDynamoStore(tableName, recordName, region string) (DynamoStore, error) {
	var dynamoOpts []func(*config.LoadOptions) error
	dynamoOpts = append(dynamoOpts, config.WithRegion(region))
	c, err := config.LoadDefaultConfig(context.Background(), dynamoOpts...)
	if err != nil {
		return DynamoStore{}, err
	}
	client := dynamodb.NewFromConfig(c)
	return DynamoStore{
		Client:    client,
		TableName: tableName,
		Record:    recordName,
	}, nil
}
func NewDynamoStoreWithClient(tableName, recordName string, client *dynamodb.Client) DynamoStore {
	return DynamoStore{
		Client:    client,
		TableName: tableName,
		Record:    recordName,
	}
}
func (s *DynamoStore) RemoveFeature(ctx context.Context, feature string) error {
	_, err := s.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{Value: s.Record},
		},
		TableName:        &s.TableName,
		UpdateExpression: aws.String("REMOVE features.#f"),
		ExpressionAttributeNames: map[string]string{
			"#f": feature,
		},
	})
	return err
}
func (s *DynamoStore) SetFeature(ctx context.Context, feature string, value bool) error {
	_, err := s.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{Value: s.Record},
		},
		TableName:        &s.TableName,
		UpdateExpression: aws.String("SET features.#f = :c"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":c": &types.AttributeValueMemberBOOL{Value: value},
		},
		ExpressionAttributeNames: map[string]string{
			"#f": feature,
		},
	})
	return err
}
func (s *DynamoStore) Load(ctx context.Context) (models.Features, map[string]models.ThrottleConfig, error) {
	gio, err := s.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.TableName,
		Key: map[string]types.AttributeValue{
			"_pk": &types.AttributeValueMemberS{Value: s.Record},
		},
	})
	if err != nil {
		return nil, nil, err
	}
	if len(gio.Item) < 1 {
		return nil, nil, errors.New("record is empty")
	}
	var f models.StoreDocument
	err = unmarshalMap(gio.Item, &f)
	if err != nil {
		return nil, nil, err
	}
	if f.Throttles == nil {
		f.Throttles = make(map[string]models.ThrottleConfig)
	}
	return f.Features, f.Throttles, nil
}

func unmarshalMap(m map[string]types.AttributeValue, out interface{}) error {
	return attributevalue.NewDecoder(func(do *attributevalue.DecoderOptions) { do.TagKey = "json" }).Decode(&types.AttributeValueMemberM{Value: m}, out)
}
