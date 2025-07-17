package features

import (
	"context"
	"path/filepath"

	"github.com/gonzaru/wmstatusbar/internal/env"
	"github.com/gonzaru/wmstatusbar/internal/util"
)

const gorumTag = "gorum"

type gorum struct{}

func init() {
	register("gorum", func() Feature { return &gorum{} })
}

func (gorum) Name() string {
	return "gorum"
}

func (gorum) Info(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	pidFile := filepath.Join(env.TmpDir, env.UserName+"-"+gorumTag+".pid")
	if !util.ProcessFromFile(pidFile) {
		return "", nil
	}

	msgFile := filepath.Join(env.TmpDir, env.UserName+"-"+gorumTag+"-wm.txt")
	msg, err := util.ReadMessageFile(msgFile)
	if err != nil {
		return "", err
	}
	return msg, nil
}
