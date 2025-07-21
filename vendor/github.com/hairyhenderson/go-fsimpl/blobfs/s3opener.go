package blobfs

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"strconv"
	"strings"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/ratelimit"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsv2cfg "github.com/aws/aws-sdk-go-v2/config"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
	typesv2 "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gocloud.dev/blob"
	"gocloud.dev/blob/s3blob"
)

// Note: most of the code in this file is taken from the Go CDK and modified
// to support anonymous access to S3 buckets.
// See https://github.com/google/go-cloud/issues/3512

var _ blob.BucketURLOpener = (*s3v2URLOpener)(nil)

type s3v2URLOpener struct {
	imdsfs fs.FS
	// Options specifies the options to pass to OpenBucket.
	Options s3blob.Options
}

const (
	sseTypeParamKey      = "ssetype"
	kmsKeyIDParamKey     = "kmskeyid"
	accelerateParamKey   = "accelerate"
	usePathStyleParamkey = "use_path_style"
	disableHTTPSParamKey = "disable_https"
)

func toServerSideEncryptionType(value string) (typesv2.ServerSideEncryption, error) {
	for _, sseType := range typesv2.ServerSideEncryptionAes256.Values() {
		if strings.EqualFold(string(sseType), value) {
			return sseType, nil
		}
	}

	return "", fmt.Errorf("%q is not a valid value for %q", value, sseTypeParamKey)
}

// OpenBucketURL opens an s3blob.Bucket based on u.
// Taken from
//
//nolint:funlen,gocyclo
func (o *s3v2URLOpener) OpenBucketURL(ctx context.Context, u *url.URL) (*blob.Bucket, error) {
	q := u.Query()

	if sseTypeParam := q.Get(sseTypeParamKey); sseTypeParam != "" {
		q.Del(sseTypeParamKey)

		sseType, err := toServerSideEncryptionType(sseTypeParam)
		if err != nil {
			return nil, err
		}

		o.Options.EncryptionType = sseType
	}

	if kmsKeyID := q.Get(kmsKeyIDParamKey); kmsKeyID != "" {
		q.Del(kmsKeyIDParamKey)

		o.Options.KMSEncryptionID = kmsKeyID
	}

	accelerate := false

	if accelerateParam := q.Get(accelerateParamKey); accelerateParam != "" {
		q.Del(accelerateParamKey)

		var err error

		accelerate, err = strconv.ParseBool(accelerateParam)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", accelerateParamKey, err)
		}
	}

	opts := []func(*s3v2.Options){
		func(o *s3v2.Options) {
			o.UseAccelerate = accelerate
		},
	}

	if disableHTTPSParam := q.Get(disableHTTPSParamKey); disableHTTPSParam != "" {
		q.Del(disableHTTPSParamKey)

		value, err := strconv.ParseBool(disableHTTPSParam)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", disableHTTPSParamKey, err)
		}

		opts = append(opts, func(o *s3v2.Options) {
			o.EndpointOptions.DisableHTTPS = value
		})
	}

	if usePathStyleParam := q.Get(usePathStyleParamkey); usePathStyleParam != "" {
		q.Del(usePathStyleParamkey)

		value, err := strconv.ParseBool(usePathStyleParam)
		if err != nil {
			return nil, fmt.Errorf("invalid value for %q: %w", usePathStyleParamkey, err)
		}

		opts = append(opts, func(o *s3v2.Options) {
			o.UsePathStyle = value
		})
	}

	cfg, err := V2ConfigFromURLParams(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("open bucket %v: %w", u, err)
	}

	if cfg.Region == "" && o.imdsfs != nil {
		// if we have an IMDS filesystem, use it to get the region
		region, err := fs.ReadFile(o.imdsfs, "meta-data/placement/region")
		if err != nil {
			return nil, fmt.Errorf("couldn't get region from IMDS: %w", err)
		}

		cfg.Region = string(region)
	}

	clientV2 := s3v2.NewFromConfig(cfg, opts...)

	return s3blob.OpenBucketV2(ctx, clientV2, u.Host, &o.Options)
}

// V2ConfigFromURLParams returns an aws.Config for AWS SDK v2 initialized based on the URL
// parameters in q. It is intended to be used by URLOpeners for AWS services if
// UseV2 returns true.
//
// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws#Config
//
// It returns an error if q contains any unknown query parameters; callers
// should remove any query parameters they know about from q before calling
// V2ConfigFromURLParams.
//
// The following query options are supported:
//   - region: The AWS region for requests; sets WithRegion.
//   - profile: The shared config profile to use; sets SharedConfigProfile.
//   - endpoint: The AWS service endpoint to send HTTP request.
//   - hostname_immutable: Make the hostname immutable, only works if endpoint is also set.
//   - dualstack: A value of "true" enables dual stack (IPv4 and IPv6) endpoints.
//   - fips: A value of "true" enables the use of FIPS endpoints.
//   - rate_limiter_capacity: A integer value configures the capacity of a token bucket used
//     in client-side rate limits. If no value is set, the client-side rate limiting is disabled.
//     See https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/retries-timeouts/#client-side-rate-limiting.
//
//nolint:funlen,gocyclo
func V2ConfigFromURLParams(ctx context.Context, q url.Values) (awsv2.Config, error) {
	var (
		endpoint          string
		hostnameImmutable bool
		rateLimitCapacity int64
		opts              []func(*awsv2cfg.LoadOptions) error
	)

	for param, values := range q {
		value := values[0]

		switch param {
		// See https://github.com/google/go-cloud/issues/3512
		case "anonymous":
			enableAnon, err := strconv.ParseBool(value)
			if err != nil {
				return awsv2.Config{}, fmt.Errorf("invalid value for anonymous: %w", err)
			}

			if enableAnon {
				opts = append(opts, awsv2cfg.WithCredentialsProvider(awsv2.AnonymousCredentials{}))
			}
		case "hostname_immutable":
			var err error

			hostnameImmutable, err = strconv.ParseBool(value)
			if err != nil {
				return awsv2.Config{}, fmt.Errorf("invalid value for hostname_immutable: %w", err)
			}
		case "region":
			opts = append(opts, awsv2cfg.WithRegion(value))
		case "endpoint":
			endpoint = value
		case "profile":
			opts = append(opts, awsv2cfg.WithSharedConfigProfile(value))
		case "dualstack":
			dualStack, err := strconv.ParseBool(value)
			if err != nil {
				return awsv2.Config{}, fmt.Errorf("invalid value for dualstack: %w", err)
			}

			if dualStack {
				opts = append(opts, awsv2cfg.WithUseDualStackEndpoint(awsv2.DualStackEndpointStateEnabled))
			}
		case "fips":
			fips, err := strconv.ParseBool(value)
			if err != nil {
				return awsv2.Config{}, fmt.Errorf("invalid value for fips: %w", err)
			}

			if fips {
				opts = append(opts, awsv2cfg.WithUseFIPSEndpoint(awsv2.FIPSEndpointStateEnabled))
			}
		case "rate_limiter_capacity":
			var err error

			rateLimitCapacity, err = strconv.ParseInt(value, 10, 32)
			if err != nil {
				return awsv2.Config{}, fmt.Errorf("invalid value for capacity: %w", err)
			}
		case "awssdk":
			// ignore, should be handled before this
		default:
			return awsv2.Config{}, fmt.Errorf("unknown query parameter %q", param)
		}
	}

	if endpoint != "" {
		//nolint:staticcheck
		customResolver := awsv2.EndpointResolverWithOptionsFunc(
			func(_, region string, _ ...any) (awsv2.Endpoint, error) {
				//nolint:staticcheck
				return awsv2.Endpoint{
					PartitionID:       "aws",
					URL:               endpoint,
					SigningRegion:     region,
					HostnameImmutable: hostnameImmutable,
				}, nil
			})
		//nolint:staticcheck
		opts = append(opts, awsv2cfg.WithEndpointResolverWithOptions(customResolver))
	}

	var rateLimiter retry.RateLimiter

	rateLimiter = ratelimit.None
	if rateLimitCapacity > 0 {
		rateLimiter = ratelimit.NewTokenRateLimit(uint(rateLimitCapacity))
	}

	opts = append(opts, awsv2cfg.WithRetryer(func() awsv2.Retryer {
		return retry.NewStandard(func(so *retry.StandardOptions) {
			so.RateLimiter = rateLimiter
		})
	}))

	return awsv2cfg.LoadDefaultConfig(ctx, opts...)
}
