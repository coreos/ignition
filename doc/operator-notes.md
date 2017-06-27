# Operator Notes

## HTTP Backoff and Retry

When Ignition is fetching a resource over http(s), if the resource is unavailable Ignition will continually retry to fetch the resource with an exponential backoff between requests.

For a given retry attempt, Ignition will wait 10 seconds for the server to send the response headers for the request. If response headers are not received in this time, or an HTTP 5XX error code is received, the request is cancelled, Ignition waits for the backoff, and a new request is made.

Any HTTP response code less than 500 results in the request being completed, and either the resource will be fetched or Ignition will fail.

Ignition will initially wait 100 milliseconds between failed attempts, and the amount of time to wait doubles for each failed attempt until it reaches 5 seconds.

## EC2 and IAM roles

Ignition has support for fetching files over the S3 protocol. When Ignition is running in EC2, it supports using the IAM role given to the EC2 instance to fetch protected assets from S3. If IAM credentials are not successfully fetched, Ignition will attempt to fetch the file with no credentials.
