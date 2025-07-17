package features

import (
	"context"
	"path/filepath"

	"github.com/gonzaru/wmstatusbar/internal/env"
	"github.com/gonzaru/wmstatusbar/internal/util"
)

const lruTag = "lru"

type lru struct{}

func init() {
	register("lru", func() Feature { return &lru{} })
}

func (lru) Name() string {
	return "lru"
}

func (lru) Info(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	pidFile := filepath.Join(env.TmpDir, env.UserName+"-"+lruTag+".pid")
	if !util.ProcessFromFile(pidFile) {
		return "", nil
	}

	msgFile := filepath.Join(env.TmpDir, env.UserName+"-"+lruTag+"-message.txt")
	msg, err := util.ReadMessageFile(msgFile)
	if err != nil {
		return "", err
	}
	return msg, nil
}
