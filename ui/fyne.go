package ui

import (
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// Constants
const (
	AppName       = "OrionChat"
	AppIcon       = "assets/icon.ico"

	// Window dimensions
	WindowWidth   = 300
	WindowHeight  = 350

	// Text sizes
	DefaultTextSize = 14

	CopyLabel = "Copy %s URL 📋"
	CopiedLabel = "Copied! ✓"

	DonationLabel = "Support us on"
	DonationLink = "https://trakteer.id/oristarium"
	DonationText = "Trakteer ☕"

	// Layout dimensions
	StatusBottomMargin = 16
	ButtonSpacing = 60
)

// RunUI starts the Fyne UI application
func RunUI(serverPort string, shutdown chan bool, serverStarted chan bool, wg *sync.WaitGroup) {
	// Create Fyne application in the main thread
	fyneApp := app.New()
	
	// Set application icon
	icon, err := fyne.LoadResourceFromPath(AppIcon)
	if err != nil {
		log.Printf("Failed to load icon: %v", err)
	} else {
		fyneApp.SetIcon(icon)
	}
	
	window := fyneApp.NewWindow(AppName)
	
	// Disable window maximizing
	window.SetFixedSize(true)

	// Create and configure the large icon image
	iconBig := canvas.NewImageFromFile("assets/icon-big.png")
	iconBig.SetMinSize(fyne.NewSize(100, 100))
	iconBig.Resize(fyne.NewSize(100, 100))
	iconBigContainer := container.NewCenter(iconBig)

	// Create status text with initial color
	status := canvas.NewText("Starting server...", color.Gray{Y: 200})
	status.TextStyle = fyne.TextStyle{Bold: true}
	status.TextSize = DefaultTextSize
	statusContainer := container.NewVBox(
		container.NewCenter(status),
		container.NewHBox(layout.NewSpacer()), // Add bottom margin
	)
	statusContainer.Resize(fyne.NewSize(0, StatusBottomMargin))

	// Create copy buttons with feedback
	createCopyButton := func(label, url string) *widget.Button {
		btn := widget.NewButton(fmt.Sprintf(CopyLabel, label), nil)
		btn.OnTapped = func() {
			window.Clipboard().SetContent(fmt.Sprintf("http://localhost%s%s", serverPort, url))
			originalText := btn.Text
			btn.SetText(CopiedLabel)
			
			// Reset text after 2 seconds
			go func() {
				time.Sleep(2 * time.Second)
				btn.SetText(originalText)
			}()
		}
		return btn
	}

	copyControlBtn := createCopyButton("Control", "/control")
	copyTTSBtn := createCopyButton("TTS", "/tts")
	copyDisplayBtn := createCopyButton("Display", "/display")

	// Create buttons container
	buttonsContainer := container.NewVBox(
		copyControlBtn,
		copyTTSBtn,
		copyDisplayBtn,
	)

	// Create tutorial button
	tutorialBtn := widget.NewButton("How to use?", func() {
		url := fmt.Sprintf("http://localhost%s/tutorial", serverPort)
		if err := fyne.CurrentApp().OpenURL(parseURL(url)); err != nil {
			log.Printf("Failed to open tutorial: %v", err)
		}
	})

	// Create quit button with red text
	quitBtn := widget.NewButton("Quit Application", func() {
		window.Close()
	})
	quitBtn.Importance = widget.DangerImportance

	// Create footer with donation link and copyright
	footerText := canvas.NewText(DonationLabel, color.Gray{Y: 200})
	footerText.TextSize = DefaultTextSize
	footerLink := widget.NewHyperlink(DonationText, parseURL(DonationLink))
	
	// Add copyright notice with smaller text
	copyrightText := canvas.NewText("© 2024 Oristarium. All rights reserved.", color.Gray{Y: 150})
	copyrightText.TextSize = DefaultTextSize - 4
	
	footerContainer := container.NewVBox(
		container.NewHBox(
			footerText,
			container.New(&spacingLayout{}, footerLink),
		),
		container.NewCenter(copyrightText),
	)
	footer := container.NewCenter(footerContainer)

	// Layout
	content := container.NewVBox(
		iconBigContainer,
		statusContainer,
		layout.NewSpacer(),
		buttonsContainer,
		container.NewHBox(layout.NewSpacer()),
		container.NewVBox(
			tutorialBtn,
			quitBtn,
			footer,
		),
	)

	// Set fixed height for the spacer
	content.Objects[3].Resize(fyne.NewSize(0, ButtonSpacing))

	// Add padding around the content
	paddedContent := container.NewPadded(content)
	window.SetContent(paddedContent)

	// Handle window closing
	window.SetCloseIntercept(func() {
		status.Text = "Shutting down..."
		close(shutdown)
	})

	// Set window size
	window.Resize(fyne.NewSize(WindowWidth, WindowHeight))

	// Update status when server starts
	go func() {
		select {
		case <-serverStarted:
			status.Color = color.RGBA{G: 180, A: 255}  // Green color
			status.Text = fmt.Sprintf("Server running on port %s", serverPort)
			status.Refresh()
		case <-shutdown:
			status.Color = color.RGBA{R: 180, A: 255}  // Red color
			status.Text = "Shutting down..."
			status.Refresh()
			return
		}
	}()

	// Run the GUI in the main thread
	window.ShowAndRun()
}

// Helper function to safely parse URLs
func parseURL(urlStr string) *url.URL {
	url, err := url.Parse(urlStr)
	if err != nil {
		log.Printf("Error parsing URL: %v", err)
		return nil
	}
	return url
}

type spacingLayout struct{}

func (s *spacingLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return objects[0].MinSize()
}

func (s *spacingLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	objects[0].Resize(objects[0].MinSize())
	objects[0].Move(fyne.NewPos(0, 0))
} 