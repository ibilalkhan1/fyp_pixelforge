module github.com/ibilalkhan1/fyp_pixelforge

go 1.24.2

// cimgui-go is vendored at third_party/cimgui-go so the studio build
// pins to a known snapshot (includes pre-compiled Dear ImGui static
// libs for linux/macos/windows). See docs/plans/2026-05-17-001-*.md U1.
replace github.com/AllenDang/cimgui-go => ./third_party/cimgui-go

require (
	github.com/AllenDang/cimgui-go v0.0.0-00010101000000-000000000000
	github.com/hajimehoshi/ebiten/v2 v2.9.9
	github.com/shirou/gopsutil/v4 v4.25.9
	github.com/stretchr/testify v1.11.1
	golang.org/x/image v0.31.0
)

require (
	github.com/akavel/rsrc v0.10.2 // indirect
	github.com/biessek/golang-ico v0.0.0-20250805151044-6d8ea19fb761 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ebitengine/gomobile v0.0.0-20250923094054-ea854a63cce1 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/oto/v3 v3.4.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/jackmordaunt/icns/v2 v2.2.7 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	github.com/josephspurrier/goversioninfo v1.7.0 // indirect
	github.com/jsummers/gobmp v0.0.0-20230614200233-a9de23ed2e25 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20250317134145-8bc96cf8fc35 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
