package main

import (
	"embed"
	"net/http"
	"net/url"
	"strings"

	"mp4-optimizer/internal/bridge"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/out
var assets embed.FS

// Version is injected at build time
var Version = "0.0.0"

func main() {
	// Create an instance of the app structure
	// Pass the build-time version
	if Version == "" {
		Version = "0.0.0"
	}
	app := bridge.NewApp(Version)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "MP4 FastStart Inspector",
		Width:  1024,
		Height: 768,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if strings.HasPrefix(path, "/video/") {
					// Serve local file
					filePath := strings.TrimPrefix(path, "/video/")
					if decodedPath, err := url.PathUnescape(filePath); err == nil {
						filePath = decodedPath
					}
					// Improve security here?
					// Ideally we check if it is a valid video file or path is absolute.
					// For now, simple serving.
					http.ServeFile(w, r, filePath)
					return
				}
				// Default asset serving is handled if we don't write generic handler?
				// Wait, if Handler is set, it overrides Assets?
				// No, Wails doc says Handler is a fallback or override?
				// "If defined, this handler will be called for every request. If it returns a 404, the AssetServer will attempt to find the file in the Assets."
				// So we should return 404 if not /video/
				w.WriteHeader(http.StatusNotFound)
			}),
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
