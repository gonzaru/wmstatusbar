package features

import (
	"context"
	"os"
	"regexp"
	"strconv"
)

var reVideo = regexp.MustCompile(`^uvcvideo\s+\d+\s+(\d+)`)

type camera struct{}

func init() {
	register("camera", func() Feature { return &camera{} })
}

func (camera) Name() string {
	return "camera"
}

func (camera) Info(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	msg := "cam: off"
	data, err := os.ReadFile("/proc/modules")
	if err != nil {
		return "", err
	}
	mVideo := reVideo.FindSubmatch(data)
	if len(mVideo) < 2 {
		return msg, nil
	}
	num, err := strconv.Atoi(string(mVideo[1]))
	if err != nil {
		return "", err
	}
	if num > 0 {
		msg = "cam: on"
	}
	return msg, nil
}
