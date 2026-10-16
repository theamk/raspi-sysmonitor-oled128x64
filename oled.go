package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"syscall"
)

const (
	i2cSlave = 0x0703
	width    = 128
	height   = 64
	chunk    = 16 // data bytes per I2C transaction
)

type oled struct {
	f    *os.File
	flip bool
}

func openOLED(bus, addr int, flip bool) (*oled, error) {
	f, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), i2cSlave, uintptr(addr)); errno != 0 {
		f.Close()
		return nil, errno
	}
	return &oled{f: f, flip: flip}, nil
}

func (o *oled) Close() error { return o.f.Close() }

func (o *oled) cmd(c ...byte) error {
	_, err := o.f.Write(append([]byte{0x00}, c...))
	return err
}

func (o *oled) init() error {
	segRemap, comScan := byte(0xA1), byte(0xC8)
	if o.flip {
		segRemap, comScan = 0xA0, 0xC0
	}
	seq := [][]byte{
		{0xAE},       // display off
		{0xD5, 0x80}, // clock divide
		{0xA8, 0x3F}, // multiplex 64
		{0xD3, 0x00}, // display offset
		{0x40},       // start line 0
		{0x8D, 0x14}, // charge pump on
		{0x20, 0x00}, // horizontal addressing
		{segRemap},   // segment remap
		{comScan},    // COM scan direction
		{0xDA, 0x12}, // COM pins
		{0x81, 0xCF}, // contrast
		{0xD9, 0xF1}, // precharge
		{0xDB, 0x40}, // VCOMH
		{0xA4},       // follow RAM
		{0xA6},       // normal (not inverted)
		{0xAF},       // display on
	}
	for _, s := range seq {
		if err := o.cmd(s...); err != nil {
			return err
		}
	}
	return nil
}

// show converts img (width x height) to SSD1306 page format and writes it.
func (o *oled) show(img image.Image) error {
	buf := make([]byte, width*height/8)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if color.GrayModel.Convert(img.At(x, y)).(color.Gray).Y > 127 {
				buf[(y/8)*width+x] |= 1 << (y % 8)
			}
		}
	}
	if err := o.cmd(0x21, 0, width-1, 0x22, 0, height/8-1); err != nil {
		return err
	}
	for i := 0; i < len(buf); i += chunk {
		if _, err := o.f.Write(append([]byte{0x40}, buf[i:i+chunk]...)); err != nil {
			return err
		}
	}
	return nil
}
