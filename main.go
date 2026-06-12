package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// webviewDataPath keeps WebView2 data next to the portable exe; when the app
// runs elevated the default per-user path may be unavailable, so fall back to
// a temp directory rather than failing to start.
func webviewDataPath() string {
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "data", "webview")
		if os.MkdirAll(p, 0o755) == nil {
			return p
		}
	}
	p := filepath.Join(os.TempDir(), "diagnostic-studio", "webview")
	_ = os.MkdirAll(p, 0o755)
	return p
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Diagnostic Studio",
		Width:  1280,
		Height: 832,
		MinWidth:  1080,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 244, G: 247, B: 248, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewUserDataPath: webviewDataPath(),
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
