package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"syscall"
)

const (
	i2cSlave = 0x0703
	chunk    = 16 // data bytes per I2C transaction
)

type oled struct {
	Width, Height int
	Img           *image.RGBA
	f             *os.File
	flip          bool
	display       string
}

func openOLED(bus, addr int, flip bool, display string) (*oled, error) {
	if display != "auto" && display != "ssd1306" && display != "sh1106" {
		return nil, fmt.Errorf("unsupported display %q (use auto, ssd1306 or sh1106)", display)
	}
	f, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), i2cSlave, uintptr(addr)); errno != 0 {
		f.Close()
		return nil, errno
	}
	if display == "auto" {
		display, err = detectDisplay(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("display auto-detection failed: %w; select -display=ssd1306 or -display=sh1106 explicitly", err)
		}
	}
	const w, h = 128, 64
	return &oled{Width: w, Height: h, Img: image.NewRGBA(image.Rect(0, 0, w, h)), f: f, flip: flip, display: display}, nil
}

func (o *oled) Close() error { return o.f.Close() }

func detectDisplay(bus io.ReadWriter) (string, error) {
	if _, err := bus.Write([]byte{0x00}); err != nil {
		return "", fmt.Errorf("select status register: %w", err)
	}
	var status [1]byte
	if _, err := io.ReadFull(bus, status[:]); err != nil {
		return "", fmt.Errorf("read status register: %w", err)
	}
	// Observed signatures used by OneBitDisplay; clones may differ, so retain explicit overrides.
	switch status[0] & 0x0F {
	case 0x08:
		return "sh1106", nil
	case 0x03, 0x06, 0x07:
		return "ssd1306", nil
	default:
		return "", fmt.Errorf("unrecognized status 0x%02x", status[0])
	}
}

func (o *oled) cmd(c ...byte) error {
	_, err := o.f.Write(append([]byte{0x00}, c...))
	return err
}

func (o *oled) init() error {
	segRemap, comScan := byte(0xA1), byte(0xC8)
	if o.flip {
		segRemap, comScan = 0xA0, 0xC0
	}
	power := []byte{0x8D, 0x14}
	addressing := []byte{0x20, 0x00}
	precharge, vcom := byte(0xF1), byte(0x40)
	if o.display == "sh1106" {
		power = []byte{0xAD, 0x8B}
		addressing = []byte{0xB0}
		precharge, vcom = 0x22, 0x35
	}
	seq := [][]byte{
		{0xAE},       // display off
		{0xD5, 0x80}, // clock divide
		{0xA8, 0x3F}, // multiplex 64
		{0xD3, 0x00}, // display offset
		{0x40},       // start line 0
		power,
		addressing,
		{segRemap},   // segment remap
		{comScan},    // COM scan direction
		{0xDA, 0x12}, // COM pins
		{0x81, 0xCF}, // contrast
		{0xD9, precharge},
		{0xDB, vcom},
		{0xA4}, // follow RAM
		{0xA6}, // normal (not inverted)
		{0xAF}, // display on
	}
	for _, s := range seq {
		if err := o.cmd(s...); err != nil {
			return err
		}
	}
	return nil
}

func (o *oled) show() error {
	buf := make([]byte, o.Width*o.Height/8)
	for y := 0; y < o.Height; y++ {
		for x := 0; x < o.Width; x++ {
			if color.GrayModel.Convert(o.Img.At(x, y)).(color.Gray).Y > 127 {
				buf[(y/8)*o.Width+x] |= 1 << (y % 8)
			}
		}
	}
	if o.display == "sh1106" {
		for page := 0; page < o.Height/8; page++ {
			// The visible 128 columns are centered in SH1106's 132-column RAM.
			if err := o.cmd(byte(0xB0+page), 0x02, 0x10); err != nil {
				return err
			}
			if err := o.writeData(buf[page*o.Width : (page+1)*o.Width]); err != nil {
				return err
			}
		}
		return nil
	}
	if err := o.cmd(0x21, 0, byte(o.Width-1), 0x22, 0, byte(o.Height/8-1)); err != nil {
		return err
	}
	return o.writeData(buf)
}

func (o *oled) writeData(data []byte) error {
	for offset := 0; offset < len(data); offset += chunk {
		end := min(offset+chunk, len(data))
		if _, err := o.f.Write(append([]byte{0x40}, data[offset:end]...)); err != nil {
			return err
		}
	}
	return nil
}
