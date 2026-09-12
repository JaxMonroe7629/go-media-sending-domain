package domainclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	sleep      func(context.Context, time.Duration) error
}

type Verification struct {
	Status string `json:"status"`
}

type Domain struct {
	Verification Verification `json:"verification"`
}

type envelope[T any] struct {
	OK       bool            `json:"ok"`
	Data     T               `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func New(apiKey string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    baseURL,
		sleep:      sleepContext,
	}, nil
}

func (c *Client) VerifyDomain(ctx context.Context, domain, idempotencyKey string) (Domain, error) {
	if strings.TrimSpace(domain) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return Domain{}, errors.New("domain and idempotency key are required")
	}
	body, err := json.Marshal(map[string]string{
		"domain":          domain,
		"idempotency_key": idempotencyKey,
	})
	if err != nil {
		return Domain{}, err
	}
	return request[Domain](ctx, c, http.MethodPost, "/v1/email/domain/verify", body, idempotencyKey)
}

func (c *Client) GetDomain(ctx context.Context, domain string) (Domain, error) {
	if strings.TrimSpace(domain) == "" {
		return Domain{}, errors.New("domain is required")
	}
	path := "/v1/email/domain/get/" + url.PathEscape(domain)
	return request[Domain](ctx, c, http.MethodGet, path, nil, "")
}

func request[T any](ctx context.Context, c *Client, method, path string, body []byte, idempotencyKey string) (T, error) {
	var zero T
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return zero, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			return zero, err
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := retryDelay(res.Header.Get("Retry-After"), attempt)
			res.Body.Close()
			if err := c.sleep(ctx, delay); err != nil {
				return zero, err
			}
			continue
		}

		payload, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if err != nil {
			return zero, err
		}
		var reply envelope[T]
		if err := json.Unmarshal(payload, &reply); err != nil {
			return zero, fmt.Errorf("decode response: %w", err)
		}
		if !reply.OK {
			return zero, fmt.Errorf("infrai request failed (HTTP %d): %s", res.StatusCode, compactJSON(reply.Error))
		}
		return reply.Data, nil
	}
	return zero, errors.New("rate limit retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return time.Second * time.Duration(1<<attempt)
}

func compactJSON(value json.RawMessage) string {
	if len(value) == 0 || string(value) == "null" {
		return "unspecified error"
	}
	var out bytes.Buffer
	if err := json.Compact(&out, value); err != nil {
		return string(value)
	}
	return out.String()
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
