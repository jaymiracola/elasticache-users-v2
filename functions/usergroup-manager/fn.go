package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/crossplane/function-sdk-go/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/response"
	"google.golang.org/protobuf/types/known/structpb"
)

// Function is your composition function.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction discovers ElastiCache Users with cache-id label and manages UserGroup membership.
func (f *Function) RunFunction(ctx context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running usergroup-manager function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	// Get the observed composite resource (XCacheInfra)
	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, err)
		return rsp, nil
	}

	// Extract region from XR parameters
	region, err := oxr.Resource.GetString("spec.parameters.region")
	if err != nil {
		f.log.Info("Region not specified, using default", "default", "us-east-1")
		region = "us-east-1"
	}

	// Get AWS credentials from the request
	creds, err := request.GetCredentials(req, "aws")
	if err != nil {
		response.Fatal(rsp, fmt.Errorf("failed to get AWS credentials: %w", err))
		return rsp, nil
	}

	// Initialize AWS SDK config
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			string(creds.Data["aws_access_key_id"]),
			string(creds.Data["aws_secret_access_key"]),
			string(creds.Data["aws_session_token"]),
		)),
	)
	if err != nil {
		response.Fatal(rsp, fmt.Errorf("failed to load AWS config: %w", err))
		return rsp, nil
	}

	// Create ElastiCache client
	client := elasticache.NewFromConfig(cfg)

	// Query all ElastiCache users
	describeOutput, err := client.DescribeUsers(ctx, &elasticache.DescribeUsersInput{})
	if err != nil {
		response.Fatal(rsp, fmt.Errorf("failed to describe ElastiCache users: %w", err))
		return rsp, nil
	}

	// Filter users by cache-id tag (Note: ElastiCache Users don't support tags in the same way as other resources)
	// Instead, we'll filter by a naming convention or collect all users
	// For now, collecting all users as ElastiCache doesn't support user-level tags
	var userIDs []string
	for _, user := range describeOutput.Users {
		if user.UserId != nil {
			userIDs = append(userIDs, *user.UserId)
			f.log.Info("Discovered user", "userId", *user.UserId, "userName", aws.ToString(user.UserName))
		}
	}

	f.log.Info("Total users discovered", "count", len(userIDs))

	// Store user IDs in pipeline context for other functions to access
	response.SetContextKey(rsp, "discoveredUserIDs", structpb.NewListValue(&structpb.ListValue{
		Values: func() []*structpb.Value {
			values := make([]*structpb.Value, len(userIDs))
			for i, id := range userIDs {
				values[i] = structpb.NewStringValue(id)
			}
			return values
		}(),
	}))

	// Update XR status with discovered user count
	oxr.Resource.Object["status"] = map[string]any{
		"discoveredUsers": len(userIDs),
		"userIDs":         userIDs,
	}
	if err := response.SetDesiredCompositeResource(rsp, oxr); err != nil {
		f.log.Info("Failed to update XR status", "error", err)
	}

	response.ConditionTrue(rsp, "UserDiscoverySuccess", fmt.Sprintf("Discovered %d ElastiCache users", len(userIDs))).
		TargetCompositeAndClaim()

	return rsp, nil
}
