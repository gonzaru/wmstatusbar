package features

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	weatherCity = flag.String(
		"feature-weather-city",
		"",
		"city to request from wttr.in (required)",
	)
	weatherFormat = flag.String(
		"feature-weather-format",
		"%t",
		"custom format string understood by wttr.in",
	)
	weatherTimeout = flag.Int(
		"feature-weather-timeout",
		5,
		"HTTP timeout in seconds for the weather feature",
	)
)

type weather struct{}

func init() {
	register("weather", func() Feature { return &weather{} })
}

func (weather) Name() string {
	return "weather"
}

func (weather) Info(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	city := strings.TrimSpace(*weatherCity)
	if city == "" {
		return "", errors.New("weather: --feature-weather-city cannot be empty")
	}

	format := strings.TrimSpace(*weatherFormat)
	link := fmt.Sprintf("https://wttr.in/%s?format=%s", url.PathEscape(city), url.QueryEscape(format))

	timeout := time.Duration(*weatherTimeout) * time.Second
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	client := http.Client{Timeout: timeout}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather: bad status %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if errRb := res.Body.Close(); errRb != nil {
		return "", errRb
	}
	return strings.TrimSpace(string(body)), nil
}
