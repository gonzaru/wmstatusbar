package features

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var reVolume = regexp.MustCompile(`(?m)\d+%`)

type audio struct{}

func init() {
	register("audio", func() Feature { return &audio{} })
}

func (audio) Name() string {
	return "audio"
}

func (audio) Info(ctx context.Context) (string, error) {
	volCmd := exec.CommandContext(ctx, "pactl", "get-sink-volume", "@DEFAULT_SINK@")
	volOut, err := volCmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(volOut))
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return "", nil
		}
		return "", fmt.Errorf("audio: pactl get-sink-volume failed: %s: %w", outStr, err)
	}
	mVol := reVolume.FindAllString(string(volOut), 2)
	switch len(mVol) {
	case 0:
		return "", nil
	case 1:
		return "vol: " + mVol[0], nil
	default:
		if mVol[0] == mVol[1] {
			return "vol: " + mVol[0], nil
		}
		return "vol: left: " + mVol[0] + " / right: " + mVol[1], nil
	}
}
