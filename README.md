# wmstatusbar - window manager status bar

### Installation

#### 1. Clone

    $ git clone https://github.com/gonzaru/wmstatusbar.git
    $ cd wmstatusbar

#### 2. Build

    $ mkdir -p bin
    $ sh build.sh

#### 3. Copy

    # copy the file to any searchable shell $PATH, for example:
    $ sudo cp bin/wmstatusbar /usr/local/bin/

#### Usage of wmstatusbar:

* shows help

```
$ wmstatusbar -h
```

* prints the output of the default features each second to the stdout (default is the feature date)

```
$ wmstatusbar
```

* prints the load average and the date features each 5 seconds without concurrency

```
$ wmstatusbar -interval 5 -features="loadavg,date" -parallel=false
```

* prints the load average and the date each 60 seconds with a custom date/time format

```
$ wmstatusbar -interval 60 -features="loadavg,date" -feature-date-format="Mon Jan 2 15:04"
```

* prints the weather and the date of the magical city of Narva and terminates

```
$ wmstatusbar -oneshot -features="weather,date" -feature-weather-city="Narva" -output=stdout
```

* updates the output of the default features each 5 seconds to the root window (xsetroot)

```
$ wmstatusbar -interval 5 -output=xsetroot
```
