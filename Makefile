# WslToggle Makefile
#
# 目标平台为 Windows：本工具是 Windows 托盘程序。
# 需要已安装 Go 1.22+ 与 GNU make（Git Bash 自带 make 亦可）。

GO     ?= go
OUT    := dist/WslToggle.exe
RSRC   := rsrc_windows_amd64.syso

.PHONY: help build windows test fmt vet clean

help:
	@echo "可用目标："
	@echo "  make windows  构建 Windows 单文件 exe（含图标资源，输出到 dist/）"
	@echo "  make test     运行单元测试"
	@echo "  make fmt      格式化源码"
	@echo "  make vet      静态检查"
	@echo "  make clean    清理构建产物"

# 生成多尺寸应用图标 + Windows 资源文件（供 go build 自动链接）
windows: $(OUT)

$(OUT): $(RSRC) $(wildcard *.go) go.mod go.sum
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags "-s -w -H windowsgui" -o $(OUT) .

$(RSRC): app.ico
	$(GO) run github.com/akavel/rsrc@v0.10.2 -ico app.ico -o $(RSRC)

app.ico:
	$(GO) run generate_icon.go icon.go

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -f $(OUT) $(RSRC) app.ico app-preview.png
