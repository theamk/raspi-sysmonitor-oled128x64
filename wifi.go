package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

type wifiSampler struct {
	iface  string
	rx, tx uint64
}

type wifiInfo struct {
	Iface    string
	Link     float64
	RxD, TxD uint64 // bytes since previous sample
}

// firstWireless returns the first interface and link quality from /proc/net/wireless.
func firstWireless() (iface string, link float64, ok bool, err error) {
	f, err := os.Open("/proc/net/wireless")
	if errors.Is(err, fs.ErrNotExist) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 0; sc.Scan(); n++ {
		if n < 2 { // two header lines
			continue
		}
		name, rest, found := strings.Cut(sc.Text(), ":")
		fields := strings.Fields(rest)
		if !found || len(fields) < 2 {
			continue
		}
		link, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return "", 0, false, err
		}
		return strings.TrimSpace(name), link, true, nil
	}
	return "", 0, false, sc.Err()
}

func netBytes(iface string) (rx, tx uint64, err error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name, rest, found := strings.Cut(sc.Text(), ":")
		if !found || strings.TrimSpace(name) != iface {
			continue
		}
		// Fields: 8 receive counters (bytes first), then 8 transmit counters.
		fields := strings.Fields(rest)
		if len(fields) < 9 {
			return 0, 0, fmt.Errorf("unexpected /proc/net/dev line for %s", iface)
		}
		if rx, err = strconv.ParseUint(fields[0], 10, 64); err != nil {
			return 0, 0, err
		}
		tx, err = strconv.ParseUint(fields[8], 10, 64)
		return rx, tx, err
	}
	if err := sc.Err(); err != nil {
		return 0, 0, err
	}
	return 0, 0, fmt.Errorf("interface %s not found in /proc/net/dev", iface)
}

// Sample returns ok=false when there is no wireless interface.
func (w *wifiSampler) Sample() (info wifiInfo, ok bool, err error) {
	iface, link, ok, err := firstWireless()
	if err != nil || !ok {
		w.iface = ""
		return wifiInfo{}, false, err
	}
	rx, tx, err := netBytes(iface)
	if err != nil {
		return wifiInfo{}, false, err
	}
	info = wifiInfo{Iface: iface, Link: link}
	// First sample, interface change or counter reset: report no traffic.
	if w.iface == iface && rx >= w.rx && tx >= w.tx {
		info.RxD, info.TxD = rx-w.rx, tx-w.tx
	}
	w.iface, w.rx, w.tx = iface, rx, tx
	return info, true, nil
}

// humanBytes formats n in at most 4 characters, e.g. 999B, 1.2K, 15M.
func humanBytes(n uint64) string {
	v, unit := float64(n), "B"
	for _, u := range []string{"K", "M", "G"} {
		if v < 1000 {
			break
		}
		v /= 1024
		unit = u
	}
	switch {
	case unit == "B":
		return fmt.Sprintf("%d%s", n, unit)
	case v < 10:
		return fmt.Sprintf("%.1f%s", v, unit)
	default:
		return fmt.Sprintf("%.0f%s", v, unit)
	}
}
