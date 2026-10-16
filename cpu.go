package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type cpuSampler struct{ total, idle uint64 }

// cpuTemp returns the CPU temperature in degrees C from hwmon0 (reported in 1/1000 C).
func cpuTemp() (float64, error) {
	b, err := os.ReadFile("/sys/class/hwmon/hwmon0/temp1_input")
	if err != nil {
		return 0, err
	}
	milli, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0, err
	}
	return float64(milli) / 1000, nil
}

func readCPU() (total, idle uint64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return 0, 0, fmt.Errorf("/proc/stat is empty")
	}
	fields := strings.Fields(sc.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("unexpected /proc/stat line: %q", sc.Text())
	}
	// Fields: user nice system idle iowait irq softirq steal ...; guest times are already in user/nice.
	for i, s := range fields[1:] {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		if i >= 8 {
			break
		}
		total += v
		if i == 3 || i == 4 {
			idle += v
		}
	}
	return total, idle, nil
}

func newCPUSampler() (*cpuSampler, error) {
	t, i, err := readCPU()
	if err != nil {
		return nil, err
	}
	return &cpuSampler{t, i}, nil
}

// Percent returns CPU busy percentage since the previous call.
func (c *cpuSampler) Percent() (float64, error) {
	t, i, err := readCPU()
	if err != nil {
		return 0, err
	}
	dt, di := t-c.total, i-c.idle
	c.total, c.idle = t, i
	if dt == 0 {
		return 0, nil
	}
	return 100 * float64(dt-di) / float64(dt), nil
}
