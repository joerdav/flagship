package flagship_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

func newTestClient() (*dynamodb.Client, string, error) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}
	creds := credentials.NewStaticCredentialsProvider("fake", "accessKeyId", "secretKeyId")
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("eu-pluto-1"),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, "", fmt.Errorf("aws-sdk: failed to create local dynamo client - %w", err)
	}
	testClient := dynamodb.NewFromConfig(cfg, dynamodb.WithEndpointResolver(dynamodb.EndpointResolverFromURL(endpoint)))
	return testClient, "eu-pluto-1", nil
}

func createLocalTable(t *testing.T, d *dynamodb.Client) string {
	t.Helper()
	name := uuid.New().String()
	_, err := d.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String("_pk"),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String("_pk"),
			KeyType:       types.KeyTypeHash,
		}},
		TableName:              &name,
		BillingMode:            types.BillingModePayPerRequest,
		GlobalSecondaryIndexes: nil,
		LocalSecondaryIndexes:  nil,
		ProvisionedThroughput:  nil,
		SSESpecification:       nil,
		StreamSpecification:    nil,
		TableClass:             "",
		Tags:                   nil,
	})
	if err != nil {
		t.Fatalf("failed to create local table: %v", err)
	}
	return name
}

func deleteLocalTable(t *testing.T, d *dynamodb.Client, name string) {
	t.Helper()
	_, err := d.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		t.Fatalf("failed to delete table: %v", err)
	}
}
