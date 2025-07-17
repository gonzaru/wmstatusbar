package features

import (
	"context"
	"flag"
	"time"
)

var dateFormat = flag.String(
	"feature-date-format",
	"Mon Jan 2 15:04:05",
	"shows the current date/time with a custom format",
)

type date struct{}

func init() {
	register("date", func() Feature { return &date{} })
}

func (date) Name() string {
	return "date"
}

func (date) Info(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	return time.Now().Format(*dateFormat), nil
}
