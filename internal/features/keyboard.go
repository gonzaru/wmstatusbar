package features

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type keyboard struct{}

var (
	keyboardVariant = flag.Bool(
		"feature-keyboard-variant",
		false,
		"shows the current keyboard layout variant",
	)
	reLayout  = regexp.MustCompile(`(?m)^layout:\s+(\S+)\s*$`)
	reVariant = regexp.MustCompile(`(?m)^variant:\s+(\S+)\s*$`)
)

func init() {
	register("keyboard", func() Feature { return &keyboard{} })
}

func (keyboard) Name() string {
	return "keyboard"
}

func (keyboard) Info(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	cmd := exec.CommandContext(ctx, "setxkbmap", "-query")
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return "", nil
		}
		return "", fmt.Errorf("keyboard: setxkbmap failed: %s: %w", outStr, err)
	}
	mLayout := reLayout.FindSubmatch(out)
	if len(mLayout) < 2 {
		return "", nil
	}
	msg := string(bytes.TrimSpace(mLayout[1]))
	if *keyboardVariant {
		if mVariant := reVariant.FindSubmatch(out); len(mVariant) >= 2 {
			msg += " " + string(bytes.TrimSpace(mVariant[1]))
		}
	}
	return msg, nil
}
