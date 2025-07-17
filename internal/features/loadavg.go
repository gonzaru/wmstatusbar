package features

import (
	"context"
	"os"
	"strings"
)

type loadavg struct{}

func init() {
	register("loadavg", func() Feature { return &loadavg{} })
}

func (loadavg) Name() string {
	return "loadavg"
}

func (loadavg) Info(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return "", nil
	}
	return "load average: " + strings.Join(fields[:3], ", "), nil
}
