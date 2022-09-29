package feature

import (
	"bytes"
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

func TestGetRun(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		features         any
		expectError      bool
		expectedFeatures any
		expectedOut      string
	}{
		{
			name: "no args",
			features: map[string]any{
				"features": map[string]any{},
			},
			expectError: true,
			expectedOut: `usage: flagship feature get [featureName]
	Returns the status of all feature flags.
	Optionally tableName and recordName can be provided. (default=featureFlagStore, features)` + "\n",
		},
		{
			name: "no features",
			features: map[string]any{
				"features": map[string]any{},
			},
			args:        []string{"aFeature"},
			expectError: true,
		},
		{
			name: "non matching feature",
			features: map[string]any{
				"features": map[string]any{
					"bFeature": true,
				},
			},
			args:        []string{"aFeature"},
			expectError: true,
		},
		{
			name: "matching feature true",
			features: map[string]any{
				"features": map[string]any{
					"aFeature": true,
				},
			},
			args:        []string{"aFeature"},
			expectedOut: "aFeature: true\n",
		},
		{
			name: "matching feature false",
			features: map[string]any{
				"features": map[string]any{
					"aFeature": false,
				},
			},
			args:        []string{"aFeature"},
			expectedOut: "aFeature: false\n",
		},
		{
			name: "matching feature true, with others",
			features: map[string]any{
				"features": map[string]any{
					"aFeature": true,
					"bFeature": false,
					"cFeature": true,
				},
			},
			args:        []string{"aFeature"},
			expectedOut: "aFeature: true\n",
		},
		{
			name: "matching feature false, with others",
			features: map[string]any{
				"features": map[string]any{
					"aFeature": false,
					"bFeature": true,
					"cFeature": false,
				},
			},
			args:        []string{"aFeature"},
			expectedOut: "aFeature: false\n",
		},
	}
	name, dclient, close := dynamotesting.CreateLocalTable(t)
	defer close()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			record := uuid.NewString()
			store := dynamostore.NewDynamoStoreWithClient(name, record, dclient)
			c := Get{Store: store, Out: out}
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
				t.Errorf("Get{}.Run(...) = %v", err)
			}
			if tt.expectError && err == nil {
				t.Errorf("Get{}.Run(...) = nil")
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
			if diff := cmp.Diff(tt.features, res); diff != "" {
				t.Error(diff)
			}
			if diff := cmp.Diff(tt.expectedOut, out.String()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
