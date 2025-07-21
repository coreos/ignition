package awssmpfs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SSMClient is an interface that wraps basic functionality of the AWS SSM
// client that is used by this filesystem. This interface is usually implemented
// by [github.com/aws/aws-sdk-go-v2/service/ssm.Client].
type SSMClient interface {
	GetParameter(ctx context.Context,
		params *ssm.GetParameterInput,
		optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParametersByPath(ctx context.Context,
		params *ssm.GetParametersByPathInput,
		optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}
