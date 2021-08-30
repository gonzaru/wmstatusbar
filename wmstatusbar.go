// by Gonzaru
// Distributed under the terms of the GNU General Public License v3

package main

// #cgo LDFLAGS: -lX11
// #include <X11/Xlib.h>
import "C"

import (
	"bytes"
	"io/ioutil"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Bar struct {
	display *C.Display
}

// audio gets the main volume from pulseaudio
func (b *Bar) audio() string {
	// checks if main audio is muted
	isMuted := func() bool {
		var mute = false
		content, errEc := exec.Command("pactl", "list", "sinks").Output()
		if errEc != nil {
			log.Fatal(errEc)
		}
		if strings.Contains(string(content), "Mute: yes") {
			mute = true
		}
		return mute
	}
	var volume = ""
	content, errEc := exec.Command("pacmd", "list-sinks").Output()
	if errEc != nil {
		log.Fatal(errEc)
	}
	match := false
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `* index: `) {
			match = true
			continue
		}
		if match && strings.Contains(line, "volume: ") {
			front := strings.Split(line, "/")
			frontLeft := strings.TrimSpace(front[1])
			frontRight := strings.TrimSpace(front[3])
			if frontLeft != "" && frontLeft == frontRight {
				volume = "vol: " + frontLeft
			}
			break
		}
	}
	if isMuted() {
		volume += " (muted)"
	}
	return volume
}

// date gets current date and time
func (b *Bar) date() string {
	timeNow := time.Now().Format("Mon Jan 2 15:04:05")
	return timeNow
}

// keyboard gets the current keyboard layout
func (b *Bar) keyboard() string {
	var layout string
	content, err := exec.Command("setxkbmap", "-query").Output()
	if err != nil {
		log.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^layout:(.*)`)
	if match := re.FindSubmatch(content); len(match) >= 2 {
		layout = string(bytes.ToUpper(bytes.TrimSpace(match[1])))
	}
	return layout
}

// loadavg gets the average system load
func (b *Bar) loadavg() string {
	var status = ""
	content, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		log.Fatal(err)
	}
	load := strings.Join(strings.Fields(string(content))[0:3], ", ")
	status = "load avg: " + load
	return status
}

// microphone tells if the default microphone is active or not
func (b *Bar) microphone() string {
	var status = "mic: off"
	content, err := exec.Command("pacmd", "list-sources").Output()
	if err != nil {
		log.Fatal(err)
	}
	match := false
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, `* index: `) {
			match = true
			continue
		}
		if match && strings.Contains(line, "muted: no") {
			status = "mic: on"
			break
		}
	}
	return status
}

// status gets the status bar
func (b *Bar) status() string {
	const maxChannels = 6
	var statusLine = ""

	channels := make(map[int]chan string)
	for i := 0; i < maxChannels; i++ {
		channels[i] = make(chan string)
	}

	// display order from left(0) to right(N)
	go func() { channels[0] <- b.microphone() }()
	go func() { channels[1] <- b.webcam() }()
	go func() { channels[2] <- b.audio() }()
	go func() { channels[3] <- b.loadavg() }()
	go func() { channels[4] <- b.keyboard() }()
	go func() { channels[5] <- b.date() }()

	messages := make([]string, maxChannels)
	for i := 0; i < maxChannels; i++ {
		select {
		case message := <-channels[i]:
			messages[i] = message
		case <-time.After(time.Second * 5):
			log.Fatal("status: error: timeout")
		}
	}
	statusLine = " " + strings.Join(messages, " | ")
	return statusLine
}

// webcam tells if an existing webcam is active or not
func (b *Bar) webcam() string {
	var status = "cam: off"
	content, errRf := ioutil.ReadFile("/proc/modules")
	if errRf != nil {
		log.Fatal(errRf)
	}
	re := regexp.MustCompile(`(?m)^uvcvideo\s[0-9]+\s([0-9]+)`)
	if match := re.FindSubmatch(content); len(match) >= 2 {
		num, errSa := strconv.Atoi(string(match[1]))
		if errSa != nil {
			log.Fatal(errSa)
		}
		if num > 0 {
			status = "cam: on"
		}
	}
	return status
}

// xsetroot sets X root window
func (b *Bar) xsetroot(status string) {
	C.XStoreName(b.display, C.XDefaultRootWindow(b.display), C.CString(status))
	C.XSync(b.display, 0)
}

// main prints the window manager status bar each second
func main() {
	var dsp *C.Display = C.XOpenDisplay(nil)
	if dsp == nil {
		log.Fatal("main: error: cannot open display")
	}
	b := Bar{display: dsp}
	for {
		b.xsetroot(b.status())
		time.Sleep(time.Second)
	}
}
