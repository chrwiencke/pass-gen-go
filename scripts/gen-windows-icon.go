//go:build ignore

package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
)

func main() {
	out := flag.String("out", "cmd/gopass/gopass.ico", "output .ico path")
	flag.Parse()

	sizes := []int{16, 32, 48, 256}
	images := make([][]byte, 0, len(sizes))
	for _, size := range sizes {
		images = append(images, makeKeyIconPNG(size))
	}

	var buf bytes.Buffer
	writeLE(&buf, uint16(0))          // reserved
	writeLE(&buf, uint16(1))          // icon
	writeLE(&buf, uint16(len(sizes))) // image count
	imageOffset := 6 + len(sizes)*16  // ICONDIR + ICONDIRENTRY records
	for i, size := range sizes {
		widthByte := byte(size)
		heightByte := byte(size)
		if size >= 256 {
			widthByte = 0
			heightByte = 0
		}
		buf.WriteByte(widthByte)
		buf.WriteByte(heightByte)
		buf.WriteByte(0) // color count
		buf.WriteByte(0) // reserved
		writeLE(&buf, uint16(1))
		writeLE(&buf, uint16(32))
		writeLE(&buf, uint32(len(images[i])))
		writeLE(&buf, uint32(imageOffset))
		imageOffset += len(images[i])
	}
	for _, img := range images {
		buf.Write(img)
	}

	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}

func writeLE(buf *bytes.Buffer, v any) {
	if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
		log.Fatal(err)
	}
}

func makeKeyIconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)

	key := color.RGBA{245, 190, 60, 255}
	shadow := color.RGBA{132, 88, 0, 255}
	scale := func(v int) int {
		scaled := v * size / 32
		if scaled < 1 {
			return 1
		}
		return scaled
	}

	drawFilledCircle(img, scale(11), scale(16), scale(8), shadow)
	drawFilledCircle(img, scale(11), scale(16), scale(4), color.RGBA{})
	fillRect(img, scale(17), scale(14), scale(30), scale(19), shadow)
	fillRect(img, scale(23), scale(18), scale(27), scale(24), shadow)
	fillRect(img, scale(27), scale(18), scale(31), scale(22), shadow)

	drawFilledCircle(img, scale(10), scale(15), scale(8), key)
	drawFilledCircle(img, scale(10), scale(15), scale(4), color.RGBA{})
	fillRect(img, scale(16), scale(13), scale(29), scale(18), key)
	fillRect(img, scale(22), scale(17), scale(26), scale(23), key)
	fillRect(img, scale(26), scale(17), scale(30), scale(21), key)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{c}, image.Point{}, draw.Src)
}

func drawFilledCircle(img *image.RGBA, centerX, centerY, radius int, c color.RGBA) {
	for y := centerY - radius; y <= centerY+radius; y++ {
		for x := centerX - radius; x <= centerX+radius; x++ {
			dx := x - centerX
			dy := y - centerY
			if dx*dx+dy*dy <= radius*radius && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}
