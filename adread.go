// Read one or more A/D channels on a TS-4200/8160 board.
package main

import (
	"apl.uw.edu/mikek/tsadc"
	"encoding/csv"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Default A/D channel configuration
var defcfg = `
channels:
  - name: Ain3
    cnum: 3
    units: volts
    c: [0., 1.]
  - name: Ain4
    cnum: 4
    units: volts
    c: [0., 1.]
  - name: Ain5
    cnum: 5
    units: volts
    c: [0., 1.]
  - name: Ain6
    cnum: 6
    units: volts
    c: [0., 1.]
`

type Channel struct {
	Name  string
	Cnum  uint
	Units string
	C     []float32 ",flow"
}

type AdcCfg struct {
	Channels []Channel
}

func timestamp(t time.Time) (int64, int) {
	tt := t.Truncate(time.Microsecond)
	return tt.Unix(), tt.Nanosecond() / 1000
}

func write_header(w io.Writer, names []string) error {
	out := csv.NewWriter(w)
	vals := []string{"seconds", "microseconds"}
	err := out.Write(append(vals, names...))
	if err == nil {
		out.Flush()
	}
	return out.Error()
}

func write_record(w io.Writer, t time.Time, data []float32) error {
	rec := make([]string, 0, len(data)+2)
	out := csv.NewWriter(w)
	secs, usecs := timestamp(t)
	rec = append(rec, strconv.FormatInt(secs, 10))
	rec = append(rec, strconv.FormatInt(int64(usecs), 10))
	for _, val := range data {
		rec = append(rec, strconv.FormatFloat(float64(val), 'f', 3, 32))
	}
	err := out.Write(rec)
	if err == nil {
		out.Flush()
	}
	return out.Error()
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [cfgfile]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Sample A/Ds and write to stdout\n\n")
		flag.PrintDefaults()
	}
	s_interval := flag.Duration("interval", time.Second,
		"A/D sampling interval")
	sys_ts4800 := flag.Bool("ts4800", false, "Configure for TS-4800 CPU board")

	flag.Parse()
	args := flag.Args()

	var contents []byte
	var err error

	if len(args) >= 1 {
		contents, err = ioutil.ReadFile(args[0])
		if err != nil {
			panic(err)
		}
	} else {
		contents = []byte(defcfg)
	}

	cfg := AdcCfg{}
	err = yaml.Unmarshal(contents, &cfg)
	if err != nil {
		panic(err)
	}

	// Channel numbers
	channels := make([]uint, len(cfg.Channels))
	// Channel names
	names := make([]string, len(cfg.Channels))
	// A/D voltage values
	x := make([]float32, len(cfg.Channels))
	// Physical values
	y := make([]float32, len(cfg.Channels))

	for i, c := range cfg.Channels {
		names[i] = c.Name
		channels[i] = c.Cnum
	}

	var adc *tsadc.Adc

	// Initialize the A/D interface
	if *sys_ts4800 {
		adc, err = tsadc.NewTs4800Adc(channels, 16, 0)
	} else {
		adc, err = tsadc.NewTs4200Adc(channels, 16, 0)
	}

	if err != nil {
		panic(err)
	}

	// Initialize signal handler
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGHUP, syscall.SIGPIPE)

	// Create a sampling function
	fsample := func(t time.Time) {
		for i, c := range cfg.Channels {
			x[i], err = adc.ReadVolts(c.Cnum)
			// Apply the calibration coefficients
			//  y = C[0] + x*(C[1] + x*(C[2] + ...))
			y[i] = 0.
			for j := len(c.C) - 1; j > 0; j-- {
				y[i] = x[i] * (c.C[j] + y[i])
			}
			y[i] += c.C[0]
			if err != nil {
				panic(err)
			}
		}
		err = write_record(os.Stdout, t, y)
		if err != nil {
			panic(err)
		}
	}

	write_header(os.Stdout, names)
	// Delay a bit before the first sample
	time.Sleep(250 * time.Millisecond)
	fsample(time.Now())

	// Create a ticker to drive the sampling goroutine
	ticker := time.NewTicker(*s_interval)
	go func() {
		for t := range ticker.C {
			fsample(t)
		}
	}()

	// Wait for signal
	<-sigs
	// Stop the sampling goroutine
	ticker.Stop()
}
