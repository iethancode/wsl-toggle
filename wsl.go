package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
)

// wslInstalledDistros 返回本机已安装的发行版名（不含默认标记）。
func wslInstalledDistros() []string {
	out := runWsl("-l", "-q")
	names := parseLines(out)
	sort.Strings(names)
	return names
}

// wslRunningDistros 返回当前正在运行的发行版名。
func wslRunningDistros() []string {
	out := runWsl("-l", "--running", "-q")
	return parseLines(out)
}

// wslIsRunning 判断指定发行版是否处于 Running 状态。
func wslIsRunning(name string) bool {
	for _, d := range wslRunningDistros() {
		if equalFold(d, name) {
			return true
		}
	}
	return false
}

// pickDefaultDistro 选出要控制的发行版：优先系统默认（带 * 标记），
// 其次任意一个已安装发行版；都没有则返回空串。
func pickDefaultDistro(installed []string) string {
	def := wslDefaultDistroName()
	if def != "" && containsFold(installed, def) {
		return def
	}
	if len(installed) > 0 {
		return installed[0]
	}
	return ""
}

// wslDefaultDistroName 从 wsl -l -v 的输出中解析带 * 的默认发行版。
func wslDefaultDistroName() string {
	out := runWsl("-l", "-v")
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "*") {
			continue
		}
		// 形如 "* Ubuntu-26.04    Running    2"
		rest := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

// keep-alive 进程：在 WSL 内挂一个永不退出的进程钉住 WSL2 虚拟机。
// WSL2 在最后一个进程退出后约 60 秒会自动回收，只跑一次性命令无法保持运行；
// 只要本托盘程序开着且未点关闭，这个进程就一直持有该发行版。
var (
	keepAliveMu  sync.Mutex
	keepAliveCmd *exec.Cmd
)

// wslStartDistro 启动发行版并挂上 keep-alive，使其持续运行不被空闲回收。
func wslStartDistro(name string) {
	keepAliveMu.Lock()
	defer keepAliveMu.Unlock()

	stopKeepAliveLocked() // 先停掉旧的（可能是切换了发行版）

	// 永不退出的常驻进程；用 sh 循环 sleep，兼容各发行版（busybox 也支持）
	cmd := exec.Command("wsl.exe", "-d", name, "-e", "sh", "-c", "while true; do sleep 86400; done")
	hideWindow(cmd)
	cmd.Env = append(cmd.Environ(), "WSL_UTF8=1")
	if err := cmd.Start(); err == nil {
		keepAliveCmd = cmd
		go func() { _ = cmd.Wait() }() // 回收，避免僵尸进程
	}
}

// wslTerminate 停掉 keep-alive 后仅终止指定发行版，不影响其它发行版。
func wslTerminate(name string) {
	stopKeepAlive()
	runWslHidden("--terminate", name)
}

// wslShutdown 停掉 keep-alive 后关闭全部发行版并回收 WSL2 轻量 VM。
func wslShutdown() {
	stopKeepAlive()
	runWslHidden("--shutdown")
}

// stopKeepAlive 结束当前 keep-alive 进程（若有）。
func stopKeepAlive() {
	keepAliveMu.Lock()
	defer keepAliveMu.Unlock()
	stopKeepAliveLocked()
}

// stopKeepAliveLocked 在已持有 keepAliveMu 时调用。
func stopKeepAliveLocked() {
	if keepAliveCmd != nil && keepAliveCmd.Process != nil {
		_ = keepAliveCmd.Process.Kill()
	}
	keepAliveCmd = nil
}

// ---------- 命令执行 ----------

// runWsl 执行 wsl 命令并捕获 stdout（用于读取发行版列表）。
// wsl.exe 走管道时常输出 UTF-16LE，统一交给 decodeWsl 处理。
func runWsl(args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	hideWindow(cmd)
	cmd.Env = append(cmd.Environ(), "WSL_UTF8=1")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// 错误不捕获，避免发行版为空等情况把正常输出吞掉
	_ = cmd.Run()

	return decodeWsl(stdout.Bytes())
}

// runWslHidden 执行一条动作型 wsl 命令（启动/终止/关机），等待其完成。
// 这些命令本身耗时以秒计，调用方已在 goroutine 中，故同步 Run 不阻塞 UI。
func runWslHidden(args ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	hideWindow(cmd)
	cmd.Env = append(cmd.Environ(), "WSL_UTF8=1")
	_ = cmd.Run()
}

// hideWindow 让子进程不弹控制台黑框。
func hideWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// CREATE_NO_WINDOW
	cmd.SysProcAttr.CreationFlags |= 0x08000000
}

// ---------- 输出解析 ----------

// decodeWsl 把 wsl.exe 的输出统一转成 Go string。
// 兼容 UTF-8（设了 WSL_UTF8）与 UTF-16LE（旧版 wsl 的默认管道编码）。
func decodeWsl(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	// UTF-16LE BOM
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		return utf16leToString(b[2:])
	}
	// UTF-8 BOM
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return string(b[3:])
	}
	// 含大量 NUL 字节 → 判定为无 BOM 的 UTF-16LE
	if looksLikeUTF16(b) {
		return utf16leToString(b)
	}
	// 兜底去掉可能残留的 NUL
	return strings.ToValidUTF8(string(bytes.ReplaceAll(b, []byte{0}, nil)), "")
}

// looksLikeUTF16 抽样判断是否为 UTF-16LE（ASCII 文本表现为奇数位为 0）。
func looksLikeUTF16(b []byte) bool {
	if len(b) < 2 || len(b)%2 != 0 {
		return false
	}
	nuls := 0
	limit := len(b)
	if limit > 64 {
		limit = 64
	}
	for i := 1; i < limit; i += 2 {
		if b[i] == 0 {
			nuls++
		}
	}
	return nuls*2 > limit/2
}

func utf16leToString(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, binary.LittleEndian.Uint16(b[i:i+2]))
	}
	return string(utf16.Decode(u))
}

// parseLines 把输出切成非空行列表。
func parseLines(s string) []string {
	var res []string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "*" {
			res = append(res, trimmed)
		}
	}
	return res
}

// equalFold 不区分大小写比较（WSL 发行版名大小写不敏感）。
func equalFold(a, b string) bool { return strings.EqualFold(a, b) }

// containsFold 判断列表是否忽略大小写地含某项。
func containsFold(list []string, s string) bool {
	if s == "" {
		return false
	}
	for _, v := range list {
		if equalFold(v, s) {
			return true
		}
	}
	return false
}
