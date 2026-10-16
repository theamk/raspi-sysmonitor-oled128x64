package main

import (
	"flag"
	"image"
	"image/draw"
	"log"
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

func render(d *oled, lines ...string) {
	draw.Draw(d.Img, d.Img.Bounds(), image.Black, image.Point{}, draw.Src)
	for i, s := range lines {
		drawLine(d.Img, i, s)
	}
	if err := d.show(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	runFor := flag.Duration("duration", 0, "exit after this long (0 = run forever)")
	flag.Parse()

	lock, err := acquireLock("/tmp/sysmonitor.lock")
	if err != nil {
		log.Fatal(err)
	}
	defer lock.Close()

	d, err := openOLED(1, 0x3C, true)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	if err := d.init(); err != nil {
		log.Fatal(err)
	}

	render(d, "Loading...")
	time.Sleep(2 * time.Second)

	var deadline <-chan time.Time
	if *runFor > 0 {
		deadline = time.After(*runFor)
	}

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		render(d, time.Now().Format("15:04:05"))
		select {
		case <-tick.C:
		case <-deadline:
			return
		}
	}
}
