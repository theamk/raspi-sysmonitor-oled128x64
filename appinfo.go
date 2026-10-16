package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"time"
)

const appInfoMax = 4096

type appInfoSource struct {
	Path  string
	Stale time.Duration // age after which the info is shown as stale
}

type appInfo struct {
	Lines [2]string // first two lines, truncated to cols
	Age   time.Duration
}

// read returns ok=false when the main app's info file does not exist.
func (s appInfoSource) read(cols int) (info appInfo, ok bool, err error) {
	f, err := os.Open(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return appInfo{}, false, nil
	}
	if err != nil {
		return appInfo{}, false, err
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return appInfo{}, false, err
	}
	// Flock is released when f is closed.

	data, err := io.ReadAll(io.LimitReader(f, appInfoMax))
	if err != nil {
		return appInfo{}, false, err
	}
	fi, err := f.Stat()
	if err != nil {
		return appInfo{}, false, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return appInfo{}, false, fmt.Errorf("no ctime available for %s", s.Path)
	}
	ctime := time.Unix(int64(st.Ctim.Sec), int64(st.Ctim.Nsec))

	parts := strings.Split(string(data), "\n")
	for i := 0; i < len(info.Lines) && i < len(parts); i++ {
		r := []rune(strings.TrimRight(parts[i], "\r"))
		if len(r) > cols {
			r = r[:cols]
		}
		info.Lines[i] = string(r)
	}
	info.Age = time.Since(ctime)
	return info, true, nil
}

// shortAge formats d in a few characters, e.g. 5m, 3h, 2d.
func shortAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// appLines returns the two bottom display lines.
func (s appInfoSource) lines(cols int) (string, string) {
	info, ok, err := s.read(cols)
	switch {
	case err != nil:
		return "app info error", ""
	case !ok:
		return "no main app", ""
	case info.Age > s.Stale:
		return info.Lines[0], "stale " + shortAge(info.Age)
	}
	return info.Lines[0], info.Lines[1]
}
