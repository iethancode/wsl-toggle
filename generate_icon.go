//go:build ignore

// 此文件仅用于生成应用图标，不参与 WslToggle 正常编译。
// 用法: go run generate_icon.go icon.go
package main

import (
	"encoding/binary"
	"image/color"
	"image/png"
	"os"
)

func main() {
	status := color.RGBA{R: 45, G: 211, B: 111, A: 255}
	sizes := []int{256, 64, 48, 32, 16}
	bodies := make([][]byte, 0, len(sizes))

	for _, size := range sizes {
		single := encodeICO(renderTerminalIcon(size, status), size)
		bodies = append(bodies, single[22:]) // 去掉单图 ICO 头，只保留 DIB 图像数据
	}

	// ICONDIR
	ico := make([]byte, 6+16*len(sizes))
	binary.LittleEndian.PutUint16(ico[2:4], 1)
	binary.LittleEndian.PutUint16(ico[4:6], uint16(len(sizes)))

	offset := uint32(len(ico))
	for i, size := range sizes {
		entry := ico[6+i*16 : 6+(i+1)*16]
		entry[0] = byte(size % 256) // 256 按 ICO 规范写为 0
		entry[1] = byte(size % 256)
		binary.LittleEndian.PutUint16(entry[4:6], 1)
		binary.LittleEndian.PutUint16(entry[6:8], 32)
		binary.LittleEndian.PutUint32(entry[8:12], uint32(len(bodies[i])))
		binary.LittleEndian.PutUint32(entry[12:16], offset)
		offset += uint32(len(bodies[i]))
	}
	for _, body := range bodies {
		ico = append(ico, body...)
	}

	if err := os.WriteFile("app.ico", ico, 0644); err != nil {
		panic(err)
	}

	// 生成 256 像素 PNG，仅用于查看设计预览。
	file, err := os.Create("app-preview.png")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if err := png.Encode(file, renderTerminalIcon(256, status)); err != nil {
		panic(err)
	}
}
