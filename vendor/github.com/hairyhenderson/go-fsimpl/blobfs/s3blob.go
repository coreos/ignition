package blobfs

import (
	"net/url"

	"github.com/hairyhenderson/go-fsimpl/internal/env"
)

// s3-specific blobfs methods

func (f *blobFS) cleanS3URL(u url.URL) url.URL {
	q := u.Query()
	translateV1Params(q)

	// allow known query parameters, remove unknown ones
	for param := range q {
		switch param {
		case "accelerate",
			"anonymous",
			"disable_https",
			"dualstack",
			"endpoint",
			"fips",
			"hostname_immutable",
			"profile",
			"rate_limiter_capacity",
			"region",
			"use_path_style":
		// not relevant for read operations, but are be passed through to the
		// Go CDK
		case "kmskeyid", "ssetype":
		default:
			q.Del(param)
		}
	}

	f.setParamsFromEnv(q)

	ensureValidEndpointURL(q)

	u.RawQuery = q.Encode()

	return u
}

// translateV1Params translates v1 query parameters to v2 query parameters.
func translateV1Params(q url.Values) {
	for param := range q {
		switch param {
		// changed to 'disable_https' in s3v2
		case "disableSSL":
			q.Set("disable_https", q.Get(param))
			q.Del(param)
		// changed to 'use_path_style' in s3v2
		case "s3ForcePathStyle":
			q.Set("use_path_style", q.Get(param))
			q.Del(param)
		}
	}
}

func ensureValidEndpointURL(q url.Values) {
	// if we have an endpoint, make sure it's a parseable URL with a scheme
	if endpoint := q.Get("endpoint"); endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil || u.Scheme == "" {
			// try adding a schema - if disable_https is set, use http, otherwise https
			if q.Get("disable_https") == "true" {
				q.Del("endpoint")
				q.Set("endpoint", "http://"+endpoint)
			} else {
				q.Del("endpoint")
				q.Set("endpoint", "https://"+endpoint)
			}
		}
	}
}

// setParamsFromEnv sets query parameters based on env vars
func (f *blobFS) setParamsFromEnv(q url.Values) {
	if q.Get("endpoint") == "" {
		endpoint := env.GetenvFS(f.envfs, "AWS_S3_ENDPOINT")
		if endpoint != "" {
			q.Set("endpoint", endpoint)
		}
	}

	if q.Get("region") == "" {
		region := env.GetenvFS(f.envfs, "AWS_REGION", env.GetenvFS(f.envfs, "AWS_DEFAULT_REGION"))
		if region != "" {
			q.Set("region", region)
		}
	}

	if q.Get("anonymous") == "" {
		anon := env.GetenvFS(f.envfs, "AWS_ANON")
		if anon != "" {
			q.Set("anonymous", anon)
		}
	}
}
