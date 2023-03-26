package data

import (
	"time"

	"github.com/avast/retry-go"
)

const RetryAttempts uint = 3
const RetryDelay time.Duration = 2 * time.Second

var CollectorDelayType = retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
	return retry.FixedDelay(n, err, config)
})

var CollectorRetryAttempts = retry.Attempts(RetryAttempts)
var CollectorRetryDelay = retry.Delay(RetryDelay)
