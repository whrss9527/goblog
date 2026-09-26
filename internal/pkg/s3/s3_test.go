package s3

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The examples of "Signature Calculations for the Authorization Header:
// Transferring Payload in a Single Chunk" in the Amazon S3 documentation.
var awsExample = &Client{Region: "us-east-1", AccessKey: "AKIAIOSFODNN7EXAMPLE", SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}

var awsExampleTime = time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC)

func TestSignAWSPutExample(t *testing.T) {
	body := []byte("Welcome to Amazon S3.")
	req, err := http.NewRequest(http.MethodPut, "https://examplebucket.s3.amazonaws.com/test%24file.text", nil)
	require.NoError(t, err)
	req.Header.Set("Date", "Fri, 24 May 2013 00:00:00 GMT")
	req.Header.Set("X-Amz-Storage-Class", "REDUCED_REDUNDANCY")
	awsExample.sign(req, body, awsExampleTime)

	assert.Equal(t, "44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072", req.Header.Get("X-Amz-Content-Sha256"))
	assert.Equal(t, "20130524T000000Z", req.Header.Get("X-Amz-Date"))
	assert.Equal(t, "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request,"+
		"SignedHeaders=date;host;x-amz-content-sha256;x-amz-date;x-amz-storage-class,"+
		"Signature=98ad721746da40c64f1a55b78f14c238d841ea1380cd77a1b5971af0ece108bd", req.Header.Get("Authorization"))
}

func TestSignAWSGetExample(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	require.NoError(t, err)
	req.Header.Set("Range", "bytes=0-9")
	awsExample.sign(req, nil, awsExampleTime)
	assert.True(t, strings.HasSuffix(req.Header.Get("Authorization"),
		"SignedHeaders=host;range;x-amz-content-sha256;x-amz-date,Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"),
		req.Header.Get("Authorization"))
}

func TestPut(t *testing.T) {
	var got struct {
		method, path, contentType, cache, auth string
		body                                   string
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got.method, got.path, got.contentType, got.cache, got.auth, got.body =
			r.Method, r.URL.EscapedPath(), r.Header.Get("Content-Type"), r.Header.Get("Cache-Control"), r.Header.Get("Authorization"), string(body)
		if strings.Contains(r.URL.Path, "denied") {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("<Error><Code>AccessDenied</Code></Error>"))
		}
	}))
	defer server.Close()

	c := &Client{Endpoint: server.URL + "/", Region: "auto", Bucket: "blog-images", AccessKey: "key", SecretKey: "secret",
		now: func() time.Time { return awsExampleTime }}
	require.NoError(t, c.Put(context.Background(), "2026/09/1758854400123.png", "image/png", []byte("png bytes")))
	assert.Equal(t, http.MethodPut, got.method)
	assert.Equal(t, "/blog-images/2026/09/1758854400123.png", got.path)
	assert.Equal(t, "image/png", got.contentType)
	assert.Equal(t, "public, max-age=31536000, immutable", got.cache)
	assert.Equal(t, "png bytes", got.body)
	assert.Contains(t, got.auth, "Credential=key/20130524/auto/s3/aws4_request")
	assert.Contains(t, got.auth, "SignedHeaders=cache-control;content-type;host;x-amz-content-sha256;x-amz-date")

	err := c.Put(context.Background(), "denied.png", "image/png", []byte("x"))
	assert.ErrorContains(t, err, "403")
	assert.ErrorContains(t, err, "AccessDenied")

	assert.Equal(t, "a%20b/%E5%9B%BE.png", escapePath("a b/图.png"))
}
