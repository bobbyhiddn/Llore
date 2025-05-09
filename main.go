package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger" // Import logger for log levels
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Llore",
		Width:  1200, // Set your desired default width here
		Height: 800,  // Set your desired default height here
		// MinWidth: 1024, // Optionally set minimum dimensions
		// MinHeight: 768, // Optionally set minimum dimensions
		LogLevel: logger.INFO, // Set log level to INFO
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		// OnDomReady:       app.domReady, // Uncomment if you have a domReady method
		OnShutdown: app.shutdown,
		// OnBeforeClose:    app.beforeClose, // Uncomment if you have a beforeClose method
		Bind: []interface{}{
			app,
		},
		// Add other options from the documentation as needed
		// Example: Frameless: false,
		// Example: StartHidden: false,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
