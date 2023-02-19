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

// bar data type
type bar struct {
	display  *C.Display
	logFile  string
	progName string
	tmpDir   string
	userName string
}

// progName the program name
const progName = "wmstatusbar"

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

// checkFeatures checks for a valid flag features
func checkFeatures() error {
	curFeatures := strings.Split(flag.Lookup("features").Value.String(), ",")
	for _, curFeature := range curFeatures {
		if curFeature == "" {
			return fmt.Errorf("checkFeatures: error: feature '%s' cannot be empy", curFeature)
		}
		if strings.Contains(curFeature, " ") {
			return fmt.Errorf("checkFeatures: error: feature '%s' cannot contain spaces", curFeature)
		}
		match := false
		for _, validFeature := range validFeatures {
			if validFeature == curFeature {
				match = true
				break
			}
		}
		if !match {
			return fmt.Errorf("checkFeatures: error: feature '%s' is not a valid feature", curFeature)
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
			return fmt.Errorf("checkFlags: error: command '%s' not found, try it without the audio feature", "pactl")
		}
	}
	if featureExists("keyboard") {
		if _, errLp := exec.LookPath("setxkbmap"); errLp != nil {
			return fmt.Errorf("checkFlags: error: command '%s' not found, try it without the keyboard feature", "setxkbmap")
		}
	}
	if featureExists("loadavg") {
		if _, err := os.Stat("/proc/loadavg"); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checkFlags: error: file '%s' does not exist, try it without the loadavg feature", "/proc/loadavg")
		}
	}
	if featureExists("camera") {
		if _, err := os.Stat("/proc/modules"); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("checkFlags: error: file '%s' does not exist, try it without the camera feature", "/proc/modules")
		}
	}
	if featureExists("weather") && flag.Lookup("feature-weather-city").Value.String() == "" {
		return errors.New("checkFlags: error: you need to use a city name for the weather feature, try it with -feature-weather-city='the city'")
	}
	return nil
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

// checkOut checks for a valid prerequisites
func checkOut() error {
	if flag.Lookup("ignoreos").Value.String() == "true" {
		if errDf := disableFeaturesOS(); errDf != nil {
			return errDf
		}
	}
	if !checkOS() {
		return fmt.Errorf("checkOut: error: '%s' has not been tested, try -ignoreos=true", runtime.GOOS)
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
	for _, curFeature := range curFeatures {
		match := false
		for _, delFeature := range delFeatures {
			if delFeature == curFeature {
				match = true
				break
			}
		}
		if !match {
			newFeatures = append(newFeatures, curFeature)
		}
	}
	if errFs := flag.Lookup("features").Value.Set(strings.Join(newFeatures, ",")); errFs != nil {
		return errFs
	}
	return nil
}

// errPrintf prints the error message to stderr according to a format specifier
func errPrintf(format string, v ...interface{}) {
	if _, err := fmt.Fprintf(os.Stderr, format, v...); err != nil {
		log.Fatal(err)
	}
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

// getUserName returns the current user name
func getUserName() string {
	usc, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usc.Username
}

// audioIsMuted checks if the main audio is muted
func (b *bar) audioIsMuted() (bool, error) {
	status := false
	content, errEc := exec.Command("pactl", "get-sink-mute", "@DEFAULT_SINK@").Output()
	if errEc != nil {
		return false, errEc
	}
	if strings.Contains(string(content), "Mute: yes") {
		status = true
	}
	return status, nil
}

// audio gets the main volume from pulseaudio
func (b *bar) audio() (string, error) {
	var statusMsg string
	if !featureExists("audio") {
		return "", nil
	}
	content, errEc := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output()
	if errEc != nil {
		return "", errEc
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
	isMuted, errAm := b.audioIsMuted()
	if errAm != nil {
		return "", errAm
	}
	if isMuted {
		statusMsg += " (muted)"
	}
	return statusMsg, nil
}

// camera tells if an existing camera is active or not
func (b *bar) camera() (string, error) {
	var statusMsg = "cam: off"
	if !featureExists("camera") {
		return "", nil
	}
	content, errRf := os.ReadFile("/proc/modules")
	if errRf != nil {
		return "", errRf
	}
	re := regexp.MustCompile(`(?m)^uvcvideo\s[0-9]+\s([0-9]+)`)
	if match := re.FindSubmatch(content); len(match) >= 2 {
		num, errSa := strconv.Atoi(string(match[1]))
		if errSa != nil {
			return "", errSa
		}
		if num > 0 {
			statusMsg = "cam: on"
		}
	}
	return statusMsg, nil
}

// date gets current date and time
func (b *bar) date() (string, error) {
	var timeNow string
	if !featureExists("date") {
		return "", nil
	}
	dateFormat := flag.Lookup("feature-date-format").Value.String()
	timeNow = time.Now().Format(dateFormat)
	return timeNow, nil
}

// gorumIsRunning checks if gorum is already running
func (b *bar) gorumIsRunning() bool {
	gorumPid := fmt.Sprintf("%s/%s-%s.pid", b.tmpDir, b.userName, "gorum")
	if _, errOsPid := os.Stat(gorumPid); errors.Is(errOsPid, os.ErrNotExist) {
		return false
	}
	contentPid, errRfPid := os.ReadFile(gorumPid)
	if errRfPid != nil {
		return false
	}
	pid, errSa := strconv.Atoi(strings.TrimRight(string(contentPid), "\n"))
	if errSa != nil {
		return false
	}
	if errSk := syscall.Kill(pid, syscall.Signal(0)); errSk != nil {
		return false
	}
	return true
}

// gorum prints gorum's media title
func (b *bar) gorum() (string, error) {
	var gorumTitle = fmt.Sprintf("%s/%s-%s-wm.txt", b.tmpDir, b.userName, "gorum")
	var title string
	if !featureExists("gorum") || !b.gorumIsRunning() {
		return "", nil
	}
	if _, errOsTitle := os.Stat(gorumTitle); errors.Is(errOsTitle, os.ErrNotExist) {
		// ignore the error if gorumTitle does not exist
		if strings.Contains(errOsTitle.Error(), "stat "+gorumTitle+": no such file or directory") {
			return "", nil
		}
		return "", errOsTitle
	}
	contentTitle, errRfTitle := os.ReadFile(gorumTitle)
	if errRfTitle != nil {
		return "", errRfTitle
	}
	title = strings.TrimRight(string(contentTitle), "\n")
	return title, nil
}

// keyboard gets the current keyboard layout
func (b *bar) keyboard() (string, error) {
	var layout, statusMsg, variant string
	if !featureExists("keyboard") {
		return "", nil
	}
	content, errSk := exec.Command("setxkbmap", "-query").Output()
	if errSk != nil {
		return "", errSk
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
	statusMsg = layout + variant
	return statusMsg, nil
}

// loadavg gets the average system load
func (b *bar) loadavg() (string, error) {
	var statusMsg string
	if !featureExists("loadavg") {
		return "", nil
	}
	content, errRf := os.ReadFile("/proc/loadavg")
	if errRf != nil {
		return "", errRf
	}
	load := strings.Join(strings.Fields(string(content))[0:3], ", ")
	statusMsg = "load avg: " + load
	return statusMsg, nil
}

// setLog sets logging output file
func (b *bar) setLog() error {
	// create the file if it does not exist or append it
	file, err := os.OpenFile(b.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	log.SetOutput(file)
	return nil
}

// status gets the status bar
func (b *bar) status() (string, error) {
	featuresStr := flag.Lookup("features").Value.String()
	if featuresStr == "" {
		return "", nil
	}
	featuresFlags := strings.Split(featuresStr, ",")
	featureSep := flag.Lookup("feature-separator").Value.String()
	channels := make(map[int]chan string)
	channelsMap := map[string]int{}
	numErrors := 0
	// displays the features from left(0) to right(N)
	for idx, feature := range featuresFlags {
		channels[idx] = make(chan string)
		channelsMap[feature] = idx
		go func(feature string) {
			var errFe error
			var statusMsg string
			switch feature {
			case "audio":
				statusMsg, errFe = b.audio()
			case "camera":
				statusMsg, errFe = b.camera()
			case "date":
				statusMsg, errFe = b.date()
			case "gorum":
				statusMsg, errFe = b.gorum()
			case "keyboard":
				statusMsg, errFe = b.keyboard()
			case "loadavg":
				statusMsg, errFe = b.loadavg()
			case "weather":
				statusMsg, errFe = b.weather()
			}
			if errFe != nil {
				log.Print(errFe.Error())
				numErrors++
			}
			channels[channelsMap[feature]] <- statusMsg
		}(feature)
	}
	numChannels := len(channels)
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
					log.Printf("status: error: timeout (5s) in channel '%s'\n", key)
					break
				}
			}
		}
	}
	if numErrors > 0 {
		errMsg := fmt.Errorf("status: error: some features contains errors, please see the log '%s'", b.logFile)
		return errMsg.Error(), errMsg
	}
	statusLine := strings.TrimRight(strings.Join(messages, ""), featureSep)
	return statusLine, nil
}

// weather gets the city weather
func (b *bar) weather() (string, error) {
	var statusMsg string
	city := flag.Lookup("feature-weather-city").Value.String()
	if city == "" {
		errMsg := errors.New("weather: error: the flag --feature-weather-city cannot be empty")
		return errMsg.Error(), errMsg
	}
	weatherFormat := flag.Lookup("feature-weather-format").Value.String()
	link := "https://wttr.in/" + city + "?format=" + weatherFormat
	res, errHg := http.Get(link)
	if errHg != nil {
		return "", errHg
	}
	content, errRa := io.ReadAll(res.Body)
	if errRa != nil {
		return "", errRa
	}
	if errBc := res.Body.Close(); errBc != nil {
		return "", errBc
	}
	contentStr := string(content)
	statusMsg = contentStr
	if strings.Contains(contentStr, "°C") {
		statusMsg = strings.Replace(contentStr, "°C", "", -1) + " °C"
	}
	return statusMsg, nil
}

// xsetroot sets X root window
func (b *bar) xsetroot(status string) {
	C.XStoreName(b.display, C.XDefaultRootWindow(b.display), C.CString(status))
	C.XSync(b.display, 0)
}

// main prints the window manager status bar
func main() {
	b := bar{
		progName: progName,
		tmpDir:   os.TempDir(),
		userName: getUserName(),
	}
	b.logFile = fmt.Sprintf("%s/%s-%s.log", b.tmpDir, b.userName, b.progName)
	if errSl := b.setLog(); errSl != nil {
		errPrintf(errSl.Error())
		log.Fatal(errSl)
	}
	var dsp *C.Display = C.XOpenDisplay(nil)
	if dsp == nil {
		errMsg := "main: error: cannot open display"
		errPrintf(errMsg)
		log.Fatal(errMsg)
	}
	b.display = dsp
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
		errPrintf(errCo.Error())
		log.Fatal(errCo)
	}
	for {
		output, errSs := b.status()
		if errSs != nil {
			log.Print(errSs)
		}
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
