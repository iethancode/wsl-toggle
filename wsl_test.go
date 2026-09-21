package main

import (
	"encoding/binary"
	"reflect"
	"testing"
	"unicode/utf16"
)

// 这些测试只验证 wsl.exe 输出解析逻辑，不依赖真实 WSL 环境，
// 因此可在全平台直接 `go test ./...` 运行（CI 上无 wsl.exe 也能通过）。

func TestParseLinesBasic(t *testing.T) {
	in := "Ubuntu-26.04\nDebian\n\nWindows\n"
	want := []string{"Ubuntu-26.04", "Debian", "Windows"}
	got := parseLines(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseLines: got %v, want %v", got, want)
	}
}

func TestParseLinesStripsCRAndBlankLines(t *testing.T) {
	// 模拟 wsl -l -q 的真实输出：纯发行版名，含 \r\n 与尾随空行
	in := "Ubuntu-26.04\r\nDebian\r\n\r\n"
	want := []string{"Ubuntu-26.04", "Debian"}
	got := parseLines(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseLines: got %v, want %v", got, want)
	}
}

func TestParseLinesOnlyStarLineIsDropped(t *testing.T) {
	// 极端情况：输出里混入一个仅含 * 的行应被丢弃，正常发行版名保留
	in := "*\nUbuntu\n"
	want := []string{"Ubuntu"}
	got := parseLines(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseLines: got %v, want %v", got, want)
	}
}

func TestParseLinesEmpty(t *testing.T) {
	if got := parseLines(""); got != nil {
		t.Errorf("parseLines(\"\") = %v, want nil", got)
	}
	if got := parseLines("\r\n"); got != nil {
		t.Errorf("parseLines(whitespace) = %v, want nil", got)
	}
}

func TestDecodeWslUTF8BOM(t *testing.T) {
	// EF BB BF + "Ubuntu"
	in := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Ubuntu")...)
	if got := decodeWsl(in); got != "Ubuntu" {
		t.Errorf("decodeWsl(utf8 bom) = %q, want %q", got, "Ubuntu")
	}
}

func TestDecodeWslUTF16LEBOM(t *testing.T) {
	// BOM FF FE + "Ubuntu" 的 UTF-16LE 编码（用 Go 自身生成，避免手写向量出错）
	u16 := make([]byte, 0, len("Ubuntu")*2)
	for _, unit := range utf16.Encode([]rune("Ubuntu")) {
		u16 = binary.LittleEndian.AppendUint16(u16, unit)
	}
	in := append([]byte{0xFF, 0xFE}, u16...)
	if got := decodeWsl(in); got != "Ubuntu" {
		t.Errorf("decodeWsl(utf16le bom) = %q, want %q", got, "Ubuntu")
	}
}

func TestDecodeWslPlainUTF8(t *testing.T) {
	if got := decodeWsl([]byte("Ubuntu-26.04")); got != "Ubuntu-26.04" {
		t.Errorf("decodeWsl(plain utf8) = %q, want %q", got, "Ubuntu-26.04")
	}
}

func TestDecodeWslEmpty(t *testing.T) {
	if got := decodeWsl(nil); got != "" {
		t.Errorf("decodeWsl(nil) = %q, want \"\"", got)
	}
}

func TestContainsFold(t *testing.T) {
	list := []string{"Ubuntu-26.04", "Debian"}
	if !containsFold(list, "ubuntu-26.04") {
		t.Error("containsFold should be case-insensitive")
	}
	if containsFold(list, "Alpine") {
		t.Error("containsFold returned true for absent item")
	}
	if containsFold(list, "") {
		t.Error("containsFold(empty) should be false")
	}
}

func TestEqualFold(t *testing.T) {
	if !equalFold("Ubuntu", "ubuntu") {
		t.Error("equalFold mismatch on case")
	}
	if equalFold("a", "b") {
		t.Error("equalFold false positive")
	}
}
