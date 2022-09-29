package feature

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/joerdav/flagship/internal/dynamostore"
	"github.com/joerdav/flagship/internal/dynamotesting"
)

func TestEnableRun(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		features         any
		expectError      bool
		expectedFeatures any
	}{
		{
			name: "no args",
			features: map[string]any{
				"features": map[string]any{},
			},
			expectedFeatures: map[string]any{
				"features": map[string]any{},
			},
			expectError: true,
		},
		{
			name: "no existing features",
			args: []string{"aFeature"},
			features: map[string]any{
				"features": map[string]any{},
			},
			expectedFeatures: map[string]any{
				"features": map[string]any{
					"aFeature": true,
				},
			},
			expectError: false,
		},
		{
			name: "feature already true",
			args: []string{"aFeature"},
			features: map[string]any{
				"features": map[string]any{
					"aFeature": true,
				},
			},
			expectedFeatures: map[string]any{
				"features": map[string]any{
					"aFeature": true,
				},
			},
			expectError: false,
		},
		{
			name: "feature false",
			args: []string{"aFeature"},
			features: map[string]any{
				"features": map[string]any{
					"aFeature": false,
				},
			},
			expectedFeatures: map[string]any{
				"features": map[string]any{
					"aFeature": true,
				},
			},
			expectError: false,
		},
	}
	name, dclient, close := dynamotesting.CreateLocalTable(t)
	defer close()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			record := uuid.NewString()
			store := dynamostore.NewDynamoStoreWithClient(name, record, dclient)
			c := Enable{Store: store}
			if tt.features != nil {
				f, err := attributevalue.MarshalMap(tt.features)
				if err != nil {
					t.Fatal(err)
				}
				f["_pk"] = &types.AttributeValueMemberS{Value: record}
				dclient.PutItem(context.Background(), &dynamodb.PutItemInput{
					Item:      f,
					TableName: &name,
				})
			}
			err := c.Run(tt.args)
			if !tt.expectError && err != nil {
				t.Errorf("Enable{}.Run(...) = %v", err)
			}
			if tt.expectError && err == nil {
				t.Errorf("Enable{}.Run(...) = nil")
			}
			i, err := dclient.GetItem(context.Background(), &dynamodb.GetItemInput{
				Key: map[string]types.AttributeValue{
					"_pk": &types.AttributeValueMemberS{Value: record},
				},
				TableName: &name,
			})
			if err != nil {
				t.Fatal(err)
			}
			var res map[string]any
			err = attributevalue.UnmarshalMap(i.Item, &res)
			if err != nil {
				t.Fatal(err)
			}
			delete(res, "_pk")
			if diff := cmp.Diff(tt.expectedFeatures, res); diff != "" {
				t.Error(diff)
			}
		})
	}
}
