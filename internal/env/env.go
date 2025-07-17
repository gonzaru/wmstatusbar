package env

import (
	"log"
	"os"
	"os/user"
)

// TmpDir is the system temporary directory
var TmpDir string

// UserName is the current user's username
var UserName string

func init() {
	TmpDir = os.TempDir()

	u, err := user.Current()
	if err != nil {
		log.Fatalf("failed to get current user: %v", err)
	}
	UserName = u.Username
}
