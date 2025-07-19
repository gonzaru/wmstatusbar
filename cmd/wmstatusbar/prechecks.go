package main

import (
	"fmt"
	"runtime"
)

// validOS lists the operating systems that have been tested
var validOS = map[string]struct{}{
	"linux": {},
}

// checkOS returns true if runtime.GOOS is in validOS
func checkOS() bool {
	_, ok := validOS[runtime.GOOS]
	return ok
}

// checkPre validates prerequisites before the program starts
func checkPre() error {
	if *flagIgnoreOS {
		return nil
	}
	if !checkOS() {
		return fmt.Errorf("unsupported OS %q; use -ignoreos=true", runtime.GOOS)
	}
	return nil
}
