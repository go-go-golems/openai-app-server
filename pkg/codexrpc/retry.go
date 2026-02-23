package codexrpc

import (
	"context"
	"errors"
	"math"
	"time"
)

const RetryableServerOverloadedCode = -32001

type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 150 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
	}
}

func normalizeRetryPolicy(policy RetryPolicy) RetryPolicy {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 1
	}
	if policy.InitialBackoff <= 0 {
		policy.InitialBackoff = 100 * time.Millisecond
	}
	if policy.MaxBackoff <= 0 {
		policy.MaxBackoff = policy.InitialBackoff
	}
	if policy.MaxBackoff < policy.InitialBackoff {
		policy.MaxBackoff = policy.InitialBackoff
	}
	return policy
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var respErr *ResponseError
	if errors.As(err, &respErr) {
		return respErr.Code == RetryableServerOverloadedCode
	}
	return false
}

func (c *Client) RequestWithRetry(ctx context.Context, method string, params any, policy RetryPolicy) (*Message, error) {
	policy = normalizeRetryPolicy(policy)

	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		resp, err := c.Request(ctx, method, params)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !IsRetryableError(err) || attempt == policy.MaxAttempts {
			break
		}

		backoff := computeBackoff(policy, attempt)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func computeBackoff(policy RetryPolicy, attempt int) time.Duration {
	if attempt <= 1 {
		return policy.InitialBackoff
	}
	multiplier := math.Pow(2, float64(attempt-1))
	backoff := time.Duration(float64(policy.InitialBackoff) * multiplier)
	if backoff > policy.MaxBackoff {
		return policy.MaxBackoff
	}
	return backoff
}
