package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gonzaru/wmstatusbar/internal/features"
)

/* flags */
var (
	featList      = flag.String("features", "date", "comma-separated list of features")
	featSeparator = flag.String("feature-separator", " | ", "string used as a separator for feature outputs")
	flagInterval  = flag.Int("interval", 1, "seconds to wait between updates")
	flagOutput    = flag.String("output", "stdout", `sends the output to "stdout", "stderr" or "xsetroot"`)
	flagOneShot   = flag.Bool("oneshot", false, "prints the status line once and terminates")
	flagParallel  = flag.Bool("parallel", true, "runs features concurrently")
	flagIgnoreOS  = flag.Bool("ignoreos", false, "does not check for the OS prerequisites")

	// go build -ldflags "-X main.version=$(git rev-parse --short HEAD)"
	version     = "dev"
	flagVersion = flag.Bool("version", false, "prints the version and exits")
)

/* usage */
func init() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(),
			`wmstatusbar – window manager status bar

USAGE:
  wmstatusbar [options]

EXAMPLES:
  wmstatusbar -interval 5 -features="loadavg,date"
  wmstatusbar -oneshot -features="weather,date" -feature-weather-city="Narva"
  wmstatusbar -oneshot -parallel=false -output=xsetroot

OPTIONS:
`)
		flag.PrintDefaults()
	}
}

/* validate flags */
func validateFlags(feats []string) error {
	switch *flagOutput {
	case "stdout", "stderr", "xsetroot":
	default:
		return fmt.Errorf(`--output must be "stdout", "stderr" or "xsetroot"`)
	}

	if *flagInterval <= 0 && !*flagOneShot {
		return fmt.Errorf("--interval must be > 0")
	}

	// has features
	has := func(feat string) bool {
		for _, f := range feats {
			if f == feat {
				return true
			}
		}
		return false
	}

	if has("weather") {
		flagCity := flag.Lookup("feature-weather-city")
		if flagCity == nil || strings.TrimSpace(flagCity.Value.String()) == "" {
			return fmt.Errorf(`feature "weather" requires --feature-weather-city`)
		}
		if flagTimeout := flag.Lookup("feature-weather-timeout"); flagTimeout != nil {
			val := strings.TrimSpace(flagTimeout.Value.String())
			intVal, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			if intVal <= 0 && strings.HasPrefix(val, "-") {
				return fmt.Errorf(`--feature-weather-timeout must be > 0`)
			}
		}
	}

	return nil
}

// main prints the window manager status bar
func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Printf("wmstatusbar version %s\n", version)
		return
	}

	if err := checkPre(); err != nil {
		log.Fatal(err)
	}

	featsActive := features.ParseList(*featList)
	if err := validateFlags(featsActive); err != nil {
		log.Printf("error: %v\n", err)
		flag.Usage()
		os.Exit(2)
	}
	reg, err := features.NewRegistry(
		featsActive,
		features.Separator(*featSeparator),
		features.WithParallel(*flagParallel),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	for {
		line, errRs := reg.Status(ctx)
		if errRs != nil {
			log.Print(errRs)
		}

		switch *flagOutput {
		case "stdout":
			fmt.Println(line)
		case "stderr":
			if _, errFf := fmt.Fprintln(os.Stderr, line); errFf != nil {
				log.Print(errFf)
			}
		case "xsetroot":
			if errEc := exec.Command("xsetroot", "-name", line).Run(); errEc != nil {
				log.Print(errEc)
			}
		}

		if *flagOneShot {
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(*flagInterval) * time.Second):
		}
	}
}
