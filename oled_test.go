package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"testing"
)

type displayProbe struct {
	*bytes.Reader
	written bytes.Buffer
}

func (probe *displayProbe) Write(data []byte) (int, error) {
	return probe.written.Write(data)
}

func TestDetectDisplay(t *testing.T) {
	for _, test := range []struct {
		name string
		data []byte
		want string
	}{
		{"sh1106", []byte{0x08}, "sh1106"},
		{"sh1106 powered", []byte{0x48}, "sh1106"},
		{"ssd1306 small", []byte{0x03}, "ssd1306"},
		{"ssd1306", []byte{0x06}, "ssd1306"},
		{"ssd1306 variant", []byte{0x07}, "ssd1306"},
		{"ssd1306 powered", []byte{0x46}, "ssd1306"},
		{"unknown", []byte{0x00}, ""},
		{"floating bus", []byte{0xFF}, ""},
		{"read failure", nil, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			probe := &displayProbe{Reader: bytes.NewReader(test.data)}
			got, err := detectDisplay(probe)
			if got != test.want || (err != nil) != (test.want == "") {
				t.Fatalf("got %q, %v; want %q", got, err, test.want)
			}
			if !bytes.Equal(probe.written.Bytes(), []byte{0}) {
				t.Fatal("probe did not select the status register")
			}
		})
	}
}

func TestOLEDShow(t *testing.T) {
	for _, display := range []string{"ssd1306", "sh1106"} {
		t.Run(display, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "oled")
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			device := oled{Width: 128, Height: 64, Img: image.NewRGBA(image.Rect(0, 0, 128, 64)), f: file, display: display}
			device.Img.Set(0, 0, image.White)
			device.Img.Set(127, 63, image.White)
			if err := device.show(); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(file.Name())
			if err != nil {
				t.Fatal(err)
			}
			var want []byte
			if display == "ssd1306" {
				want = append(want, 0x00, 0x21, 0, 127, 0x22, 0, 7)
			}
			for page := 0; page < 8; page++ {
				if display == "sh1106" {
					want = append(want, 0x00, byte(0xB0+page), 0x02, 0x10)
				}
				for column := 0; column < 128; column++ {
					if column%16 == 0 {
						want = append(want, 0x40)
					}
					pixel := byte(0)
					if page == 0 && column == 0 {
						pixel = 1
					}
					if page == 7 && column == 127 {
						pixel = 0x80
					}
					want = append(want, pixel)
				}
			}
			if !bytes.Equal(got, want) {
				t.Fatal("unexpected frame commands or pixel data")
			}
		})
	}
}

func TestOLEDInit(t *testing.T) {
	for _, display := range []string{"ssd1306", "sh1106"} {
		for _, flip := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/flip=%v", display, flip), func(t *testing.T) {
				file, err := os.CreateTemp(t.TempDir(), "oled")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				device := oled{f: file, display: display, flip: flip}
				if err := device.init(); err != nil {
					t.Fatal(err)
				}
				got, err := os.ReadFile(file.Name())
				if err != nil {
					t.Fatal(err)
				}
				power, addressing := []byte{0, 0x8D, 0x14}, []byte{0, 0x20, 0}
				if display == "sh1106" {
					power, addressing = []byte{0, 0xAD, 0x8B}, []byte{0, 0xB0}
					if bytes.Contains(got, []byte{0, 0x8D}) || bytes.Contains(got, []byte{0, 0x20}) {
						t.Fatal("SH1106 received SSD1306-only commands")
					}
				}
				orientation := []byte{0, 0xA1, 0, 0xC8}
				if flip {
					orientation = []byte{0, 0xA0, 0, 0xC0}
				}
				for _, command := range [][]byte{power, addressing, orientation, {0, 0xAF}} {
					if !bytes.Contains(got, command) {
						t.Fatalf("missing initialization command %x", command)
					}
				}
			})
		}
	}
}
