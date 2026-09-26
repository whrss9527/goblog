// Package s3 uploads objects to S3-compatible storage (Cloudflare R2, AWS S3,
// MinIO…) with AWS Signature Version 4. It implements the one call goblog
// needs, PutObject, without pulling in an SDK.
package s3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Client puts objects into one bucket.
type Client struct {
	// Endpoint is the service address without the bucket, e.g.
	// https://<account>.r2.cloudflarestorage.com or https://s3.us-east-1.amazonaws.com.
	Endpoint  string
	Region    string // "auto" for R2
	Bucket    string
	AccessKey string
	SecretKey string
	HTTP      *http.Client
	// now is replaced in tests.
	now func() time.Time
}

// Put stores body under key (path style: <endpoint>/<bucket>/<key>).
func (c *Client) Put(ctx context.Context, key, contentType string, body []byte) error {
	target, err := url.Parse(strings.TrimRight(c.Endpoint, "/") + "/" + escapePath(c.Bucket) + "/" + escapePath(key))
	if err != nil {
		return fmt.Errorf("s3: endpoint: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", contentType)
	// uploads get a new name every time: they never change
	req.Header.Set("Cache-Control", "public, max-age=31536000, immutable")
	now := time.Now
	if c.now != nil {
		now = c.now
	}
	c.sign(req, body, now().UTC())

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("s3: put %s: %w", key, err)
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		detail, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("s3: put %s: %s: %s", key, res.Status, strings.TrimSpace(string(detail)))
	}
	return nil
}

// sign adds the SigV4 headers (x-amz-date, x-amz-content-sha256, Authorization).
// Every header already on the request is signed along with host.
func (c *Client) sign(req *http.Request, body []byte, now time.Time) {
	amzDate := now.Format("20060102T150405Z")
	day := now.Format("20060102")
	payloadHash := sha256Hex(body)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	headers := map[string]string{"host": req.URL.Host}
	for name, values := range req.Header {
		headers[strings.ToLower(name)] = strings.TrimSpace(strings.Join(values, ","))
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	var canonicalHeaders strings.Builder
	for _, name := range names {
		canonicalHeaders.WriteString(name + ":" + headers[name] + "\n")
	}
	signedHeaders := strings.Join(names, ";")

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := day + "/" + c.Region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))

	key := hmacSHA256([]byte("AWS4"+c.SecretKey), day)
	key = hmacSHA256(key, c.Region)
	key = hmacSHA256(key, "s3")
	key = hmacSHA256(key, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(key, stringToSign))

	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+c.AccessKey+"/"+scope+
		",SignedHeaders="+signedHeaders+",Signature="+signature)
}

// escapePath percent-encodes an object key the way SigV4 expects: everything
// but unreserved characters, keeping the slashes.
func escapePath(key string) string {
	var b strings.Builder
	for i := 0; i < len(key); i++ {
		ch := key[i]
		switch {
		case 'A' <= ch && ch <= 'Z', 'a' <= ch && ch <= 'z', '0' <= ch && ch <= '9', ch == '-', ch == '_', ch == '.', ch == '~', ch == '/':
			b.WriteByte(ch)
		default:
			fmt.Fprintf(&b, "%%%02X", ch)
		}
	}
	return b.String()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
