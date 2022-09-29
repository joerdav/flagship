package dynamotesting

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

const region = "eu-west-1"

func CreateLocalTable(t *testing.T) (name string, testClient *dynamodb.Client, delete func()) {
	o := dynamodb.Options{
		Credentials: credentials.NewStaticCredentialsProvider("fake", "accessKeyId", "secretKeyId"),
		Region:      region,
	}
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}
	testClient = dynamodb.New(o, dynamodb.WithEndpointResolver(dynamodb.EndpointResolverFromURL(endpoint)))
	name = fmt.Sprintf("test-%s-%s", time.Now().Format("20060102-1504"), uuid.New())
	_, err := testClient.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("_pk"),
				AttributeType: "S",
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("_pk"),
				KeyType:       "HASH",
			},
		},
		BillingMode: types.BillingModePayPerRequest,
		TableName:   aws.String(name),
	})
	if err != nil {
		t.Fatalf("failed to create local table: %v", err)
	}
	delete = func() {
		_, err := testClient.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
			TableName: aws.String(name),
		})
		if err != nil {
			t.Fatalf("failed to delete table: %v", err)
		}
	}
	return
}
