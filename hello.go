package main

import (
	"image"
	"image/draw"
	"log"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func main() {
	d, err := openOLED(1, 0x3C, true)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	if err := d.init(); err != nil {
		log.Fatal(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.Black, image.Point{}, draw.Src)
	if err := d.show(img); err != nil {
		log.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	(&font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: basicfont.Face7x13,
		Dot:  fixed.P(0, 15),
	}).DrawString("Hello, World!")
	if err := d.show(img); err != nil {
		log.Fatal(err)
	}
}
