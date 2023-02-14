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
	"io"
	"log"
	"net/http"
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

// validFeatures allowed features
var validFeatures = []string{
	"audio",
	"camera",
	"date",
	"gorum",
	"keyboard",
	"loadavg",
	"weather",
}

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

// checkFeatures checks for a valid flag features
func checkFeatures() error {
	curFeatures := strings.Split(flag.Lookup("features").Value.String(), ",")
	for _, curFeature := range curFeatures {
		if curFeature == "" {
			return fmt.Errorf("error: feature '%s' cannot be empy", curFeature)
		}
		if strings.Contains(curFeature, " ") {
			return fmt.Errorf("error: feature '%s' cannot contain spaces", curFeature)
		}
		match := false
		for _, validFeature := range validFeatures {
			if validFeature == curFeature {
				match = true
				break
			}
		}
		if !match {
			return fmt.Errorf("error: feature '%s' is not a valid feature", curFeature)
		}
	}
	return nil
}

// checkFlags checks for a valid flags prerequisites
func checkFlags() error {
	if errCf := checkFeatures(); errCf != nil {
		return errCf
	}
	if featureExists("audio") {
		if _, errLp := exec.LookPath("pactl"); errLp != nil {
			return fmt.Errorf("error: command '%s' not found, try it without the audio feature", "pactl")
		}
	}
	if featureExists("keyboard") {
		if _, errLp := exec.LookPath("setxkbmap"); errLp != nil {
			return fmt.Errorf("error: command '%s' not found, try it without the keyboard feature", "setxkbmap")
		}
	}
	if featureExists("loadavg") {
		if _, err := os.Stat("/proc/loadavg"); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error: file '%s' does not exist, try it without the loadavg feature", "/proc/loadavg")
		}
	}
	if featureExists("camera") {
		if _, err := os.Stat("/proc/modules"); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error: file '%s' does not exist, try it without the camera feature", "/proc/modules")
		}
	}
	if featureExists("weather") && flag.Lookup("feature-weather-city").Value.String() == "" {
		return errors.New("error: you need to use a city name for the weather feature, try it with -feature-weather-city='the city'")
	}
	return nil
}

// checkOut checks for a valid prerequisites
func checkOut() error {
	if flag.Lookup("ignoreos").Value.String() == "true" {
		if errDf := disableFeaturesOS(); errDf != nil {
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

// disableFeaturesOS disables some OS dependent features
func disableFeaturesOS() error {
	curFeatures := strings.Split(flag.Lookup("features").Value.String(), ",")
	delFeatures := []string{"audio", "keyboard", "loadavg", "camera"}
	newFeatures := make([]string, 0)
	for _, feature := range curFeatures {
		match := false
		for _, d := range delFeatures {
			if d == feature {
				match = true
				break
			}
		}
		if !match {
			newFeatures = append(newFeatures, feature)
		}
	}
	if errFs := flag.Lookup("features").Value.Set(strings.Join(newFeatures, ",")); errFs != nil {
		return errFs
	}
	return nil
}

// featureExists checks if the feature exists in the flags
func featureExists(feature string) bool {
	features := strings.Split(flag.Lookup("features").Value.String(), ",")
	status := false
	for _, val := range features {
		if val == feature {
			status = true
			break
		}
	}
	return status
}

// audioIsMuted checks if the main audio is muted
func (b *bar) audioIsMuted() bool {
	status := false
	content, errEc := exec.Command("pactl", "get-sink-mute", "@DEFAULT_SINK@").Output()
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
	if !featureExists("audio") {
		return ""
	}
	content, errEc := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output()
	if errEc != nil {
		return errEc.Error()
	}
	line := string(content)
	if strings.Contains(line, "Volume: ") {
		front := strings.Split(line, "/")
		frontLeft := strings.TrimSpace(front[1])
		frontRight := strings.TrimSpace(front[3])
		statusMsg = "vol: left: " + frontLeft + " right: " + frontRight
		if frontLeft != "" && frontLeft == frontRight {
			statusMsg = "vol: " + frontLeft
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
	if !featureExists("camera") {
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
	if !featureExists("date") {
		return ""
	}
	dateFormat := flag.Lookup("feature-date-format").Value.String()
	timeNow = time.Now().Format(dateFormat)
	return timeNow
}

// gorum prints gorum's media title
func (b *bar) gorum() string {
	var (
		gorumPid   = fmt.Sprintf("%s/%s-%s.pid", b.tmpDir, b.userName, "gorum")
		gorumTitle = fmt.Sprintf("%s/%s-%s-wm.txt", b.tmpDir, b.userName, "gorum")
		title      string
	)
	if !featureExists("gorum") {
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
	var layout, variant string
	if !featureExists("keyboard") {
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
	if flag.Lookup("feature-keyboard-variant").Value.String() == "true" {
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
	if !featureExists("loadavg") {
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

// status gets the status bar
func (b *bar) status() string {
	featuresStr := flag.Lookup("features").Value.String()
	if featuresStr == "" {
		return ""
	}
	featureSep := flag.Lookup("feature-separator").Value.String()
	// display the order from left(0) to right(N)
	featuresFlags := strings.Split(featuresStr, ",")
	channelsMap := map[string]int{}
	for key, value := range featuresFlags {
		channelsMap[value] = key
	}
	numChannels := len(channelsMap)
	channels := make(map[int]chan string)
	for i := 0; i < numChannels; i++ {
		channels[i] = make(chan string)
	}
	if _, ok := channelsMap["date"]; ok {
		go func() { channels[channelsMap["date"]] <- b.date() }()
	}
	if _, ok := channelsMap["weather"]; ok {
		go func() { channels[channelsMap["weather"]] <- b.weather() }()
	}
	if _, ok := channelsMap["keyboard"]; ok {
		go func() { channels[channelsMap["keyboard"]] <- b.keyboard() }()
	}
	if _, ok := channelsMap["loadavg"]; ok {
		go func() { channels[channelsMap["loadavg"]] <- b.loadavg() }()
	}
	if _, ok := channelsMap["audio"]; ok {
		go func() { channels[channelsMap["audio"]] <- b.audio() }()
	}
	if _, ok := channelsMap["camera"]; ok {
		go func() { channels[channelsMap["camera"]] <- b.camera() }()
	}
	if _, ok := channelsMap["gorum"]; ok {
		go func() { channels[channelsMap["gorum"]] <- b.gorum() }()
	}
	messages := make([]string, numChannels)
	for i := 0; i < numChannels; i++ {
		select {
		case message := <-channels[i]:
			if message != "" {
				messages[i] = message + featureSep
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
	statusLine := strings.TrimRight(strings.Join(messages, ""), featureSep)
	return statusLine
}

// weather gets the city weather
func (b *bar) weather() string {
	var statusMsg string
	city := flag.Lookup("feature-weather-city").Value.String()
	if city == "" {
		return ""
	}
	weatherFormat := flag.Lookup("feature-weather-format").Value.String()
	link := "https://wttr.in/" + city + "?format=" + weatherFormat
	res, errHg := http.Get(link)
	if errHg != nil {
		errPrintf(errHg.Error())
	}
	content, errRa := io.ReadAll(res.Body)
	if errRa != nil {
		errPrintf(errRa.Error())
	}
	if errBc := res.Body.Close(); errBc != nil {
		errPrintf(errBc.Error())
	}
	contentStr := string(content)
	statusMsg = contentStr
	if strings.Contains(contentStr, "°C") {
		statusMsg = strings.Replace(contentStr, "°C", "", -1) + " °C"
	}
	return statusMsg
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
	flag.String("features", "date",
		fmt.Sprint(
			"audio: shows the main volume percentatge\n",
			"camera: shows if the camera is on/off\n",
			"date: shows the current date\n",
			"gorum: prints gorum's media title\n",
			"keyboard: shows the current keyboard layout\n",
			"loadavg: shows the system load average\n",
			"weather: shows the city weather\n",
			"* the order is shown from left to right (keyboard,date,etc.)",
		),
	)
	flag.String("feature-date-format", "Mon Jan 2 15:04:05", "shows the current date with a custom format")
	flag.Bool("feature-keyboard-variant", false, "shows the current keyboard layout and its respective variant")
	flag.String("feature-separator", " | ", "the string separator for the features")
	flag.String("feature-weather-city", "", "the city to show the weather")
	flag.String("feature-weather-format", "%t(%f)", "the city weather with a custom format (wttr.in)")
	flag.Bool("ignoreos", false, "does not check for the OS prerequisites (also it ignores some features)")
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
		if *outputFlag && output != "" {
			fmt.Println(output)
		}
		if *oneShotFlag {
			break
		}
		time.Sleep(time.Second * time.Duration(*intervalFlag))
	}
}
