// Read one or more A/D channels on a TS-4200/8160 board.
package main

import (
	"apl.uw.edu/mikek/tsadc"
	"encoding/csv"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"strconv"
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
	channels := []uint{3, 4, 5, 6}
	adc, err := tsadc.NewTs4200Adc(channels, 16, 0)
	if err != nil {
		panic(err)
	}

	cfg := AdcCfg{}
	err = yaml.Unmarshal([]byte(defcfg), &cfg)
	if err != nil {
		panic(err)
	}

	vals := make([]float32, len(cfg.Channels))
	names := make([]string, len(cfg.Channels))

	for i, c := range cfg.Channels {
		names[i] = c.Name
	}

	t := time.Now()
	for i, c := range cfg.Channels {
		vals[i], err = adc.ReadVolts(c.Cnum)
		if err != nil {
			panic(err)
		}
	}
	write_header(os.Stdout, names)
	err = write_record(os.Stdout, t, vals)
	if err != nil {
		panic(err)
	}
}
