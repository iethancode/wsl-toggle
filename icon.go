package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"sync"
)

const (
	colGreen = iota
	colRed
	colYellow
	colGray
)

var (
	iconCache   = map[int][]byte{}
	iconCacheMu sync.Mutex
)

// buildIcon 生成“终端窗口 + 状态灯”托盘图标。
// 深蓝窗口与 >_ 表示 WSL 终端，右下角状态灯表示运行状态。
func buildIcon(kind int) []byte {
	iconCacheMu.Lock()
	defer iconCacheMu.Unlock()
	if icon, ok := iconCache[kind]; ok {
		return icon
	}

	var status color.RGBA
	switch kind {
	case colGreen:
		status = color.RGBA{R: 45, G: 211, B: 111, A: 255}
	case colRed:
		status = color.RGBA{R: 244, G: 75, B: 82, A: 255}
	case colYellow:
		status = color.RGBA{R: 250, G: 190, B: 48, A: 255}
	default:
		status = color.RGBA{R: 145, G: 158, B: 177, A: 255}
	}

	icon := encodeICO(renderTerminalIcon(32, status), 32)
	iconCache[kind] = icon
	return icon
}

// renderTerminalIcon 使用 4 倍超采样绘制，再缩小以获得平滑边缘。
func renderTerminalIcon(size int, status color.RGBA) *image.RGBA {
	const ss = 4
	hiSize := size * ss
	hi := image.NewRGBA(image.Rect(0, 0, hiSize, hiSize))
	scale := float64(hiSize) / 32

	// 终端窗口：蓝色外壳、深蓝内容区。
	fillRoundedRect(hi, 2*scale, 3*scale, 29*scale, 27*scale, 5*scale,
		color.RGBA{R: 75, G: 158, B: 255, A: 255})
	fillRoundedRect(hi, 3.5*scale, 4.5*scale, 27.5*scale, 25.5*scale, 3.7*scale,
		color.RGBA{R: 20, G: 34, B: 57, A: 255})

	// 标题栏与三个窗口控制点。
	fillRect(hi, 3.5*scale, 9.2*scale, 27.5*scale, 10.3*scale,
		color.RGBA{R: 49, G: 79, B: 112, A: 255})
	fillCircle(hi, 6.1*scale, 7*scale, 1*scale, color.RGBA{R: 244, G: 97, B: 102, A: 255})
	fillCircle(hi, 9.2*scale, 7*scale, 1*scale, color.RGBA{R: 250, G: 190, B: 48, A: 255})
	fillCircle(hi, 12.3*scale, 7*scale, 1*scale, color.RGBA{R: 45, G: 211, B: 111, A: 255})

	// 命令提示符 >_。
	prompt := color.RGBA{R: 218, G: 244, B: 255, A: 255}
	drawLine(hi, 7.2*scale, 14*scale, 11.7*scale, 17.6*scale, 1.7*scale, prompt)
	drawLine(hi, 11.7*scale, 17.6*scale, 7.2*scale, 21.2*scale, 1.7*scale, prompt)
	fillRoundedRect(hi, 14*scale, 20*scale, 21*scale, 21.7*scale, .8*scale, prompt)

	// 右下角状态灯：深色描边确保在浅色和深色任务栏上都清晰。
	fillCircle(hi, 25.7*scale, 25.7*scale, 5.3*scale,
		color.RGBA{R: 10, G: 18, B: 31, A: 255})
	fillCircle(hi, 25.7*scale, 25.7*scale, 3.8*scale, status)
	fillCircle(hi, 24.5*scale, 24.3*scale, .75*scale,
		color.RGBA{R: 255, G: 255, B: 255, A: 145})

	return downsample(hi, size)
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 float64, c color.RGBA) {
	for y := int(y0); y < int(math.Ceil(y1)); y++ {
		for x := int(x0); x < int(math.Ceil(x1)); x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func fillRoundedRect(img *image.RGBA, x0, y0, x1, y1, radius float64, c color.RGBA) {
	for y := int(math.Floor(y0)); y < int(math.Ceil(y1)); y++ {
		for x := int(math.Floor(x0)); x < int(math.Ceil(x1)); x++ {
			px, py := float64(x)+.5, float64(y)+.5
			cx := math.Max(x0+radius, math.Min(px, x1-radius))
			cy := math.Max(y0+radius, math.Min(py, y1-radius))
			dx, dy := px-cx, py-cy
			if dx*dx+dy*dy <= radius*radius && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, radius float64, c color.RGBA) {
	r2 := radius * radius
	for y := int(math.Floor(cy - radius)); y <= int(math.Ceil(cy+radius)); y++ {
		for x := int(math.Floor(cx - radius)); x <= int(math.Ceil(cx+radius)); x++ {
			dx, dy := float64(x)+.5-cx, float64(y)+.5-cy
			if dx*dx+dy*dy <= r2 && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1, width float64, c color.RGBA) {
	dx, dy := x1-x0, y1-y0
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		fillCircle(img, x0+dx*t, y0+dy*t, width/2, c)
	}
}

func downsample(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	scale := src.Bounds().Dx() / size
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a uint32
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					c := src.RGBAAt(x*scale+sx, y*scale+sy)
					r += uint32(c.R)
					g += uint32(c.G)
					b += uint32(c.B)
					a += uint32(c.A)
				}
			}
			n := uint32(scale * scale)
			dst.SetRGBA(x, y, color.RGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: uint8(a / n)})
		}
	}
	return dst
}

// encodeICO 把 RGBA 图编码成 32 位单图像 ICO，供托盘运行时使用。
func encodeICO(img *image.RGBA, size int) []byte {
	pixel := size * size * 4
	maskRow := ((size + 31) / 32) * 4
	mask := maskRow * size

	var body bytes.Buffer
	_ = binary.Write(&body, binary.LittleEndian, uint32(40))
	_ = binary.Write(&body, binary.LittleEndian, int32(size))
	_ = binary.Write(&body, binary.LittleEndian, int32(size*2))
	_ = binary.Write(&body, binary.LittleEndian, uint16(1))
	_ = binary.Write(&body, binary.LittleEndian, uint16(32))
	_ = binary.Write(&body, binary.LittleEndian, uint32(0))
	_ = binary.Write(&body, binary.LittleEndian, uint32(pixel+mask))
	_ = binary.Write(&body, binary.LittleEndian, int32(0))
	_ = binary.Write(&body, binary.LittleEndian, int32(0))
	_ = binary.Write(&body, binary.LittleEndian, uint32(0))
	_ = binary.Write(&body, binary.LittleEndian, uint32(0))

	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			body.WriteByte(byte(b >> 8))
			body.WriteByte(byte(g >> 8))
			body.WriteByte(byte(r >> 8))
			body.WriteByte(byte(a >> 8))
		}
	}
	body.Write(make([]byte, mask))

	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))

	entry := make([]byte, 16)
	entry[0] = byte(size % 256)
	entry[1] = byte(size % 256)
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint16(entry[6:8], 32)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(body.Len()))
	binary.LittleEndian.PutUint32(entry[12:16], 22)
	buf.Write(entry)
	buf.Write(body.Bytes())
	return buf.Bytes()
}
