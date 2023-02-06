// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

// Package main implements the status bar
package main

// #cgo LDFLAGS: -lX11
// #include <X11/Xlib.h>
import "C"

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// bar data type
type bar struct {
	display  *C.Display
	tmpDir   string
	userName string
}

// errPrintf prints the error message to stderr according to a format specifier
func errPrintf(format string, v ...interface{}) {
	if _, err := fmt.Fprintf(os.Stderr, format, v...); err != nil {
		log.Fatal(err)
	}
}

// getUserName returns the current user name
func getUserName() string {
	usc, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usc.Username
}

// checkOS checks if the current operating system has been tested
func checkOS() bool {
	status := false
	items := []string{"linux"}
	for _, item := range items {
		if item == runtime.GOOS {
			status = true
			break
		}
	}
	return status
}

// checkFlags checks for a valid flags prerequisites
func checkFlags() error {
	if flag.Lookup("audio").Value.String() == "true" || flag.Lookup("microphone").Value.String() == "true" {
		cmds := []string{"pactl", "pacmd"}
		for _, cmd := range cmds {
			if _, errLp := exec.LookPath(cmd); errLp != nil {
				return fmt.Errorf("error: command '%s' not found, try -audio=false -microphone=false", cmd)
			}
		}
	}
	if flag.Lookup("keyboard").Value.String() == "true" {
		if _, errLp := exec.LookPath("setxkbmap"); errLp != nil {
			return fmt.Errorf("error: command '%s' not found, try -keyboard=false", "setxkbmap")
		}
	}
	if flag.Lookup("loadavg").Value.String() == "true" {
		if _, err := os.Stat("/proc/loadavg"); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error: file '%s' does not exists, try -loadavg=false", "/proc/loadavg")
		}
	}
	if flag.Lookup("camera").Value.String() == "true" {
		if _, err := os.Stat("/proc/modules"); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error: file '%s' does not exists, try -camera=false", "/proc/modules")
		}
	}
	return nil
}

// checkOut checks for a valid prerequisites
func checkOut() error {
	if flag.Lookup("ignoreos").Value.String() == "true" {
		if errDf := disableFlagsOS(); errDf != nil {
			return errDf
		}
	}
	if !checkOS() {
		return fmt.Errorf("error: '%s' has not been tested, try -ignoreos=true", runtime.GOOS)
	}
	if errCf := checkFlags(); errCf != nil {
		return errCf
	}
	return nil
}

// disableFlagsOS disables some OS dependent flags
func disableFlagsOS() error {
	flags := []string{"audio", "microphone", "keyboard", "loadavg", "camera"}
	for _, value := range flags {
		if errFs := flag.Lookup(value).Value.Set("false"); errFs != nil {
			return errFs
		}
	}
	return nil
}

// audioIsMuted checks if the main audio is muted
func (b *bar) audioIsMuted() bool {
	status := false
	content, errEc := exec.Command("pactl", "list", "sinks").Output()
	if errEc != nil {
		errPrintf(errEc.Error())
	}
	if strings.Contains(string(content), "Mute: yes") {
		status = true
	}
	return status
}

// audio gets the main volume from pulseaudio
func (b *bar) audio() string {
	var statusMsg string
	if flag.Lookup("audio").Value.String() == "false" {
		return ""
	}
	content, errEc := exec.Command("pacmd", "list-sinks").Output()
	if errEc != nil {
		return errEc.Error()
	}
	match := false
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `* index: `) {
			match = true
		} else if strings.Contains(line, " index: ") {
			match = false
		}
		if match && strings.Contains(line, "volume: ") {
			front := strings.Split(line, "/")
			frontLeft := strings.TrimSpace(front[1])
			frontRight := strings.TrimSpace(front[3])
			if frontLeft != "" && frontLeft == frontRight {
				statusMsg = "vol: " + frontLeft
			}
			break
		}
	}
	if b.audioIsMuted() {
		statusMsg += " (muted)"
	}
	return statusMsg
}

// camera tells if an existing camera is active or not
func (b *bar) camera() string {
	var statusMsg = "cam: off"
	if flag.Lookup("camera").Value.String() == "false" {
		return ""
	}
	content, errRf := os.ReadFile("/proc/modules")
	if errRf != nil {
		errPrintf(errRf.Error())
	}
	re := regexp.MustCompile(`(?m)^uvcvideo\s[0-9]+\s([0-9]+)`)
	if match := re.FindSubmatch(content); len(match) >= 2 {
		num, errSa := strconv.Atoi(string(match[1]))
		if errSa != nil {
			errPrintf(errSa.Error())
		}
		if num > 0 {
			statusMsg = "cam: on"
		}
	}
	return statusMsg
}

// date gets current date and time
func (b *bar) date() string {
	var timeNow string
	if flag.Lookup("date").Value.String() == "false" {
		return ""
	}
	if flag.Lookup("date-seconds").Value.String() == "false" {
		timeNow = time.Now().Format("Mon Jan 2 15:04")
	} else {
		timeNow = time.Now().Format("Mon Jan 2 15:04:05")
	}
	return timeNow
}

// gorum prints gorum's media title
func (b *bar) gorum() string {
	var (
		gorumPid   = fmt.Sprintf("%s/%s-%s.pid", b.tmpDir, b.userName, "gorum")
		gorumTitle = fmt.Sprintf("%s/%s-%s-wm.txt", b.tmpDir, b.userName, "gorum")
		title      string
	)
	if flag.Lookup("gorum").Value.String() == "false" {
		return ""
	}
	if _, errOsPid := os.Stat(gorumPid); errors.Is(errOsPid, os.ErrNotExist) {
		return ""
	}
	contentPid, errRfPid := os.ReadFile(gorumPid)
	if errRfPid != nil {
		return ""
	}
	pid, errSa := strconv.Atoi(strings.TrimRight(string(contentPid), "\n"))
	if errSa != nil {
		return ""
	}
	if errSk := syscall.Kill(pid, syscall.Signal(0)); errSk != nil {
		return ""
	}
	if _, errOsTitle := os.Stat(gorumTitle); errors.Is(errOsTitle, os.ErrNotExist) {
		return ""
	}
	contentTitle, errRfTitle := os.ReadFile(gorumTitle)
	if errRfTitle != nil {
		return ""
	}
	title = strings.TrimRight(string(contentTitle), "\n")
	return title
}

// keyboard gets the current keyboard layout
func (b *bar) keyboard() string {
	var (
		layout  string
		variant string
	)
	if flag.Lookup("keyboard").Value.String() == "false" {
		return ""
	}
	content, err := exec.Command("setxkbmap", "-query").Output()
	if err != nil {
		errPrintf(err.Error())
	}
	reLayout := regexp.MustCompile(`(?m)^layout:\s+([a-z][a-z])$`)
	if matchLayout := reLayout.FindSubmatch(content); len(matchLayout) >= 2 {
		layout = string(bytes.ToUpper(bytes.TrimSpace(matchLayout[1])))
	}
	if flag.Lookup("keyboard-variant").Value.String() == "true" {
		reVariant := regexp.MustCompile(`(?m)^variant:\s+(\w+)$`)
		if matchVariant := reVariant.FindSubmatch(content); len(matchVariant) >= 2 {
			variant = " " + string(bytes.TrimSpace(matchVariant[1]))
		}
	}
	return layout + variant
}

// loadavg gets the average system load
func (b *bar) loadavg() string {
	var statusMsg string
	if flag.Lookup("loadavg").Value.String() == "false" {
		return ""
	}
	content, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		errPrintf(err.Error())
	}
	load := strings.Join(strings.Fields(string(content))[0:3], ", ")
	statusMsg = "load avg: " + load
	return statusMsg
}

// microphone tells if the default microphone is active or not
func (b *bar) microphone() string {
	var statusMsg = "mic: off"
	if flag.Lookup("microphone").Value.String() == "false" {
		return ""
	}
	content, err := exec.Command("pacmd", "list-sources").Output()
	if err != nil {
		errPrintf(err.Error())
	}
	match := false
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `* index: `) {
			match = true
		} else if strings.Contains(line, " index: ") {
			match = true
		}
		if match && strings.Contains(line, "muted: no") {
			statusMsg = "mic: on"
			break
		}
	}
	return statusMsg
}

// status gets the status bar
func (b *bar) status() string {
	// display the order from left(0) to right(N)
	channelsMap := map[string]int{
		"gorum":      0,
		"microphone": 1,
		"camera":     2,
		"audio":      3,
		"loadavg":    4,
		"keyboard":   5,
		"date":       6,
	}
	numChannels := len(channelsMap)
	channels := make(map[int]chan string)
	for i := 0; i < numChannels; i++ {
		channels[i] = make(chan string)
	}
	go func() { channels[channelsMap["gorum"]] <- b.gorum() }()
	go func() { channels[channelsMap["microphone"]] <- b.microphone() }()
	go func() { channels[channelsMap["camera"]] <- b.camera() }()
	go func() { channels[channelsMap["audio"]] <- b.audio() }()
	go func() { channels[channelsMap["loadavg"]] <- b.loadavg() }()
	go func() { channels[channelsMap["keyboard"]] <- b.keyboard() }()
	go func() { channels[channelsMap["date"]] <- b.date() }()
	messages := make([]string, numChannels)
	for i := 0; i < numChannels; i++ {
		select {
		case message := <-channels[i]:
			if message != "" {
				messages[i] = message + " | "
			}
		case <-time.After(time.Second * 5):
			for key, value := range channelsMap {
				if value == i {
					errPrintf("status: error: timeout in channel '%s'\n", key)
					break
				}
			}
		}
	}
	statusLine := strings.TrimRight(strings.Join(messages, ""), " | ")
	return statusLine
}

// xsetroot sets X root window
func (b *bar) xsetroot(status string) {
	C.XStoreName(b.display, C.XDefaultRootWindow(b.display), C.CString(status))
	C.XSync(b.display, 0)
}

// main prints the window manager status bar
func main() {
	var output string
	var dsp *C.Display = C.XOpenDisplay(nil)
	if dsp == nil {
		log.Fatal("main: error: cannot open display")
	}
	intervalFlag := flag.Int("interval", 1, "seconds to wait between updates")
	oneShotFlag := flag.Bool("oneshot", false, "executes the program once and terminates")
	outputFlag := flag.Bool("output", false, "prints the output to stdout")
	rootWindowFlag := flag.Bool("rootwindow", true, "updates the root window's name")
	flag.Bool("audio", true, "shows the main volume percentage")
	flag.Bool("camera", true, "shows if the camera is on/off")
	flag.Bool("date", true, "shows the current date")
	flag.Bool("date-seconds", true, "shows the current seconds in date")
	flag.Bool("ignoreos", false, "does not check for the OS prerequisites (also it ignores some flags)")
	flag.Bool("gorum", false, "prints gorum's media title")
	flag.Bool("keyboard", true, "shows the keyboard layout")
	flag.Bool("keyboard-variant", true, "shows the keyboard layout variant")
	flag.Bool("loadavg", true, "shows the system load average")
	flag.Bool("microphone", true, "shows if the microphone is on/off")
	flag.Parse()
	if errCo := checkOut(); errCo != nil {
		log.Fatal(errCo)
	}
	b := bar{
		display:  dsp,
		tmpDir:   os.TempDir(),
		userName: getUserName(),
	}
	for {
		output = b.status()
		if *rootWindowFlag {
			b.xsetroot(output)
		}
		if *outputFlag {
			fmt.Println(output)
		}
		if *oneShotFlag {
			break
		}
		time.Sleep(time.Second * time.Duration(*intervalFlag))
	}
}
