# WslToggle

> [中文](README.md) · [English](README.en.md)

A Windows system-tray utility for controlling WSL. Right-click the tray icon to 🟢 start / 🔴 stop a WSL distro, with the icon color reflecting the live state.

Compiles to a **single `.exe`** with no runtime dependencies — no .NET, Go, or Python needed on the target machine. Double-click and it just works, as long as WSL is enabled there.

![](app-preview.png)

## Features

- **🟢 Start WSL** — boots the selected distro and holds a resident keep-alive process so the WSL2 VM stays up.
- **🔴 Stop current distro** — `wsl --terminate` on just the selected distro; other distros untouched.
- **⛔ Stop all WSL** — `wsl --shutdown` closes every distro and reclaims the WSL2 VM.
- **📦 Choose distro** — auto-scans all installed distros, pre-selects the system default, switch anytime.

Tray icon reflects state and refreshes every 2 s:

| Icon | Meaning |
|------|---------|
| 🟢 green | selected distro running |
| 🔴 red | selected distro stopped |
| 🟡 yellow | starting / stopping in progress |
| ⚪ gray | no WSL distro detected |

## Why the keep-alive?

WSL2 reclaims the VM ~60 s after the **last process exits**. If you just tap `wsl` in a cmd window and close it, WSL goes with it.

So "Start" isn't just a light toggle — it spawns a never-exiting process inside the distro (`sh -c 'while true; do sleep 86400; done'`). As long as this tray app is running and you haven't stopped it, WSL stays up. "Stop" (or quitting the app) releases that hold and WSL reclaims normally.

## Usage

### Build from source

Requires Go 1.22+ on Windows.

```powershell
.\build.ps1
```

The script: tidies deps → generates a multi-size app icon and embeds it as a `.syso` resource → cross-compiles a console-free single-file exe. Output lands in `.\dist\WslToggle.exe`.

Equivalently with the Makefile (needs GNU make, e.g. Git Bash):

```bash
make windows
```

The build workflow also runs as GitHub Actions CI/CD — pushing a `v*` tag publishes a release with the exe (see `.github/workflows/build.yml`).

## How it works

- `wsl -l -q` / `wsl -l --running -q` / `wsl -l -v` enumerate installed / running / default distros.
- Handles both **UTF-8** (`WSL_UTF8=1`) and **UTF-16LE** (legacy default) `wsl.exe` pipe output, so names with unusual characters don't garble.
- All `wsl.exe` calls use `CREATE_NO_WINDOW` — no console flash.
- Icons are drawn in memory (terminal window + prompt + status light, 4× supersampled) and encoded to ICO; no image assets ship with the binary.

## Project layout

```
main.go              tray UI, menus, 2s poll loop, keep-alive lifecycle
wsl.go               wsl command wrappers + encoding + distro parsing
icon.go              in-memory tray icon generation (4 states)
generate_icon.go     builds app.ico / app-preview.png (build-time, build-tagged)
build.ps1            one-shot build script (Windows)
Makefile             cross-platform build entry
wsl_test.go          unit tests for the parsing logic
.github/workflows    CI: vet/test + publish Windows exe
```

## Testing

Parsing logic (line splitting, default-distro detection, UTF-16/UTF-8 decode) is unit-tested and runs without a real WSL:

```bash
go test ./...
```

## License

[MIT](LICENSE)
