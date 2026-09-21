package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

// 全局受控状态，用 mu 保护。
var (
	mu       sync.Mutex
	selected string   // 当前控制的发行版名
	distros  []string // 启动时扫描到的发行版列表
	busy     bool     // 正在执行启动/关闭时暂停轮询
)

// 子菜单项与其发行版名一一对应，用于切换勾选。
type distroItem struct {
	name string
	item *systray.MenuItem
}

var distroItems []distroItem

func main() {
	// 启动前扫描一次，决定初始选中项（优先系统默认发行版）
	distros = wslInstalledDistros()
	selected = pickDefaultDistro(distros)

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(buildIcon(colGray))
	systray.SetTitle("WSL")
	systray.SetTooltip("WSL 控制器")

	mStart := systray.AddMenuItem("🟢 启动 WSL", "启动选中的 WSL 发行版")
	mStop := systray.AddMenuItem("🔴 关闭当前发行版", "终止选中的发行版")
	mStopAll := systray.AddMenuItem("⛔ 关闭全部 WSL", "关闭所有发行版并回收 WSL2 虚拟机")
	systray.AddSeparator()

	mChoose := systray.AddMenuItem("📦 选择发行版", "切换要控制的发行版")
	buildDistroSubmenu(mChoose)

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出 WSL 控制器")

	refreshStatus()
	go pollLoop()

	for {
		select {
		case <-mStart.ClickedCh:
			go actStart()
		case <-mStop.ClickedCh:
			go actStopCurrent()
		case <-mStopAll.ClickedCh:
			go actStopAll()
		case <-mQuit.ClickedCh:
			stopKeepAlive() // 退出时放开对 WSL 的持有
			systray.Quit()
			return
		}
	}
}

func onExit() {}

// buildDistroSubmenu 依据启动扫描结果生成选择子菜单。
// getlantern/systray 不支持动态移除子菜单，故列表在程序生命周期内构建一次；
// 之后新增/删除的发行版通过“重启程序”刷新，选中态则实时更新勾选。
func buildDistroSubmenu(mChoose *systray.MenuItem) {
	distroItems = nil

	if len(distros) == 0 {
		item := mChoose.AddSubMenuItem("未检测到 WSL 发行版", "")
		item.Disable()
		return
	}

	for _, name := range distros {
		d := name
		item := mChoose.AddSubMenuItem(d, "切换到 "+d)
		if equalFold(d, getSelected()) {
			item.Check()
		}
		distroItems = append(distroItems, distroItem{name: d, item: item})
		go func(item *systray.MenuItem, d string) {
			for range item.ClickedCh {
				selectDistro(d)
			}
		}(item, d)
	}
}

func pollLoop() {
	for {
		time.Sleep(2 * time.Second)
		if !isBusy() {
			refreshStatus()
		}
	}
}

// refreshStatus 查询选中发行版运行状态并更新托盘图标/文字。
func refreshStatus() {
	name := getSelected()
	if name == "" {
		systray.SetIcon(buildIcon(colGray))
		systray.SetTooltip("WSL 控制器 - 未检测到发行版")
		return
	}

	if wslIsRunning(name) {
		systray.SetIcon(buildIcon(colGreen))
		systray.SetTooltip(fmt.Sprintf("WSL: %s - 运行中", name))
	} else {
		systray.SetIcon(buildIcon(colRed))
		systray.SetTooltip(fmt.Sprintf("WSL: %s - 已关闭", name))
	}
}

func actStart() {
	name := getSelected()
	if name == "" {
		systray.SetTooltip("WSL 控制器 - 没有可启动的发行版")
		return
	}
	setBusy(true)
	defer func() { setBusy(false); refreshStatus() }()

	showBusy(fmt.Sprintf("WSL: %s - 启动中…", name))
	wslStartDistro(name)
}

func actStopCurrent() {
	name := getSelected()
	if name == "" {
		return
	}
	setBusy(true)
	defer func() { setBusy(false); refreshStatus() }()

	showBusy(fmt.Sprintf("WSL: %s - 关闭中…", name))
	wslTerminate(name)
}

func actStopAll() {
	setBusy(true)
	defer func() { setBusy(false); refreshStatus() }()

	showBusy("WSL - 全部关闭中…")
	wslShutdown()
}

// showBusy 立即把图标切成黄色，给出“处理中”即时反馈，避免误显示成已完成状态。
func showBusy(tip string) {
	systray.SetIcon(buildIcon(colYellow))
	systray.SetTooltip(tip)
}

func selectDistro(d string) {
	setSelected(d)
	for _, di := range distroItems {
		if di.name == d {
			di.item.Check()
		} else {
			di.item.Uncheck()
		}
	}
	refreshStatus()
}

// ---------- 受保护的状态存取 ----------

func getSelected() string {
	mu.Lock()
	defer mu.Unlock()
	return selected
}

func setSelected(v string) {
	mu.Lock()
	defer mu.Unlock()
	selected = v
}

func isBusy() bool {
	mu.Lock()
	defer mu.Unlock()
	return busy
}

func setBusy(v bool) {
	mu.Lock()
	defer mu.Unlock()
	busy = v
}
