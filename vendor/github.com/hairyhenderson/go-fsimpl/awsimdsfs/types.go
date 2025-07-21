package awsimdsfs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

type IMDSClient interface {
	GetDynamicData(ctx context.Context, params *imds.GetDynamicDataInput, optFns ...func(*imds.Options)) (*imds.GetDynamicDataOutput, error)
	GetMetadata(ctx context.Context, params *imds.GetMetadataInput, optFns ...func(*imds.Options)) (*imds.GetMetadataOutput, error)
	GetUserData(ctx context.Context, params *imds.GetUserDataInput, optFns ...func(*imds.Options)) (*imds.GetUserDataOutput, error)
}
