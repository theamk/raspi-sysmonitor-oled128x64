package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"log"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

func drawLine(img draw.Image, line int, s string) {
	// Inconsolata 8x16 has a 13 px ascent and 16 px line height.
	(&font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: inconsolata.Regular8x16,
		Dot:  fixed.P(0, 13+16*line),
	}).DrawString(s)
}

// justify puts right at the end of a cols-wide line, with left at the start.
func justify(left, right string, cols int) string {
	if pad := cols - len(left) - len(right); pad > 0 {
		return left + strings.Repeat(" ", pad) + right
	}
	return left + " " + right
}

func render(d *oled, lines ...string) {
	draw.Draw(d.Img, d.Img.Bounds(), image.Black, image.Point{}, draw.Src)
	for i, s := range lines {
		drawLine(d.Img, i, s)
	}
	if err := d.show(); err != nil {
		log.Fatal(err)
	}
}

func wifiLine(w *wifiSampler, cols int) string {
	info, ok, err := w.Sample()
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		return "NO WIFI"
	}
	return justify(fmt.Sprintf("Q%.0f", info.Link),
		fmt.Sprintf("R%s T%s", humanBytes(info.RxD), humanBytes(info.TxD)), cols)
}

func main() {
	runFor := flag.Duration("duration", 0, "exit after this long (0 = run forever)")
	flip := flag.Bool("flip", true, "rotate the display 180 degrees")
	showVersion := flag.Bool("version", false, "print app name and build info, then exit")
	logEvery := flag.Duration("log-interval", 10*time.Minute, "how often to log the displayed lines to stderr after the first render (0 = only once)")
	app := appInfoSource{}
	flag.StringVar(&app.Path, "appinfo-file", "/dev/shm/app-info", "main app status file")
	flag.DurationVar(&app.Stale, "appinfo-timeout", 60*time.Second, "age after which the app status is shown as stale")
	flag.Parse()
	if *showVersion {
		fmt.Println(versionString())
		return
	}
	log.Printf("starting: flip=%v log-interval=%v appinfo-file=%s appinfo-timeout=%v", *flip, *logEvery, app.Path, app.Stale)

	lock, err := acquireLock("/tmp/sysmonitor.lock")
	if err != nil {
		log.Fatal(err)
	}
	defer lock.Close()

	d, err := openOLED(1, 0x3C, *flip)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	if err := d.init(); err != nil {
		log.Fatal(err)
	}

	cpu, err := newCPUSampler()
	if err != nil {
		log.Fatal(err)
	}

	render(d, "Loading...")
	time.Sleep(2 * time.Second)

	var wifi wifiSampler
	if _, _, err := wifi.Sample(); err != nil { // baseline for traffic deltas
		log.Fatal(err)
	}

	var deadline <-chan time.Time
	if *runFor > 0 {
		deadline = time.After(*runFor)
	}

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	cols := d.Width / 8 // Inconsolata 8x16 glyph width
	var lastLog time.Time
	for {
		load, err := cpu.Percent()
		if err != nil {
			log.Fatal(err)
		}
		app1, app2 := app.lines(cols)
		lines := []string{
			justify(fmt.Sprintf("CPU %.0f%%", load), time.Now().Format("15:04:05"), cols),
			wifiLine(&wifi, cols),
			app1,
			app2,
		}
		render(d, lines...)
		if lastLog.IsZero() || (*logEvery > 0 && time.Since(lastLog) >= *logEvery) {
			log.Printf("display: %q", lines)
			lastLog = time.Now()
		}
		select {
		case <-tick.C:
		case <-deadline:
			return
		}
	}
}
