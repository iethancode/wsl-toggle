# WslToggle

> [中文](README.md) · [English](README.en.md)

一个驻留 Windows 系统托盘的 WSL 控制小工具。右键托盘图标即可 🟢 启动 / 🔴 关闭 WSL 发行版，并实时用图标颜色反映运行状态。

编译成**单个 `.exe`，无需安装 .NET / Go / Python 等任何运行时**，直接双击即用，发给别的 Windows 电脑也能跑（前提是那台电脑已启用 WSL）。

![](app-preview.png)

## 功能

- **🟢 启动 WSL**：拉起选中的发行版，并挂一个常驻 keep-alive 进程钉住 WSL2 虚拟机。
- **🔴 关闭当前发行版**：只终止选中的发行版（`wsl --terminate`），不影响其他。
- **⛔ 关闭全部 WSL**：`wsl --shutdown` 关闭所有发行版并回收整个 WSL2 虚拟机。
- **📦 选择发行版**：自动扫描本机所有 WSL 发行版，默认选中系统默认项，可随时切换。
- **自动识别**：启动时自动发现本机安装与默认发行版，无需手填。

托盘图标四态，每 2 秒刷新一次：

| 图标 | 含义 |
|------|------|
| 🟢 绿 | 选中发行版运行中 |
| 🔴 红 | 选中发行版已关闭 |
| 🟡 黄 | 正在启动/关闭 |
| ⚪ 灰 | 未检测到 WSL 发行版 |

## 为什么需要 keep-alive

WSL2 在**最后一个进程退出后约 60 秒**会自动回收虚拟机。如果你只在 cmd 里临时敲一下 `wsl`、窗口一关就没了，WSL 也会跟着关。

本工具的“启动”不只是点亮 WSL，而是在发行版里挂一个永不退出的常驻进程（`sh -c 'while true; do sleep 86400; done'`），只要托盘程序开着且没点关闭，WSL 就持续运行。点“关闭”或退出托盘程序时会把这个进程放开，WSL 按正常流程回收。

## 使用

### 直接运行（免编译）

仓库不提交编译产物。若你只想要现成的 exe，在本地执行一次构建脚本即可：

```powershell
.\build.ps1
```

产物在 `.\dist\WslToggle.exe`，把它拷到任何 64 位 Windows 10/11 机器上双击运行。

### 从源码构建

要求：

- Go 1.22+
- Windows（目标平台）

```powershell
.\build.ps1
```

脚本会：整理依赖 → 生成多尺寸应用图标并嵌入 `.syso` 资源 → 交叉编译为无控制台的单文件 exe。

等价的手工命令：

```powershell
go mod tidy
go run generate_icon.go icon.go
go run github.com/akavel/rsrc@v0.10.2 -ico app.ico -o rsrc_windows_amd64.syso
$env:CGO_ENABLED = "0"; $env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w -H windowsgui" -o dist\WslToggle.exe .
```

或者用 Makefile（需要 Git Bash）：

```bash
make windows
```

## 工作原理

- 用 `wsl -l -q` / `wsl -l --running -q` / `wsl -l -v` 分别枚举已安装、运行中、默认发行版。
- 兼容 `wsl.exe` 管道输出的两种编码：**UTF-8**（设了 `WSL_UTF8=1`）与 **UTF-16LE**（旧版默认），避免乱码与空字符。
- 所有调用 `wsl.exe` 都用 `CREATE_NO_WINDOW`，不闪黑框。
- 图标在内存中现画（终端窗口 + 命令提示符 + 状态灯，4 倍超采样），编码成 ICO，不依赖任何图片素材。

## 项目结构

```
main.go              托盘 UI、菜单、状态轮询、keep-alive 生命周期
wsl.go               wsl 命令封装 + 编码处理 + 发行版解析
icon.go              内存生成托盘图标（多态彩色）
generate_icon.go     生成应用图标 app.ico / app-preview.png（构建用，带 build tag）
build.ps1            一键构建脚本（Windows）
Makefile             跨平台构建入口
wsl_test.go          解析逻辑单元测试
.github/workflows    CI：vet/test + 发布 Windows exe
```

## 测试

解析逻辑（行切分、默认发行版识别、UTF-16/UTF-8 解码等）有单元测试，无需真实 WSL 即可运行：

```bash
go test ./...
```

## 自动化构建

`.github/workflows/build.yml` 配置了 GitHub Actions：推 `main`/`master` 时跑 vet/test 并上传构建产物，打 `v*` tag 时自动发 GitHub Release（含 Windows 单文件 exe）。

## 许可证

[MIT](LICENSE)
