# wmstatusbar - window manager status bar

### Installation

#### 1. Clone

    $ git clone https://github.com/gonzaru/wmstatusbar.git
    $ cd wmstatusbar

#### 2. Build

    $ go build -ldflags "-s -w" -o bin/wmstatusbar wmstatusbar.go

#### 3. Copy

    # copy the file to any searchable shell $PATH, for example:
    $ sudo cp bin/wmstatusbar /usr/local/bin/

#### Usage of wmstatusbar:

* shows help

```
$ wmstatusbar -h
```

* without any arguments, prints the output of the default features each second to the root window (default is only date)

```
$ wmstatusbar
```

* updates the output of the default features each 10 seconds to the root window

```
$ wmstatusbar -interval=10
```

* prints only the audio and loadavg features

```
$ wmstatusbar -features="audio,loadavg"
```

* prints the date and the weather of the magical city of Narva

```
$ wmstatusbar -features="date,weather" -feature-weather-city="Narva"
```

* prints the audio and date once and terminates (useful to combine it with another tool)

```
$ wmstatusbar -features="audio,date" -oneshot=true -rootwindow=false -output=true
```
