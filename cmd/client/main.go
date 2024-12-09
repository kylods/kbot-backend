package main

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
)

const version string = "INDEV"

var serverURL string = "http://localhost:8080"

type queueEntryWidget struct {
	widget.Label
	UUID uuid.UUID
}

type queueEntryData struct {
	Title    string
	Duration uint
	QueuedBy string
	UUID     uuid.UUID
}

var queueDataArray []queueEntryData

func fetchQueueData() {
	return
}

func newQueueEntry() *queueEntryWidget {
	qe := &queueEntryWidget{}
	qe.ExtendBaseWidget(qe)
	return qe
}

func (qe *queueEntryWidget) TappedSecondary(pe *fyne.PointEvent) {
	menu := fyne.NewMenu(
		"Right Menu",
		fyne.NewMenuItem("Remove from queue", func() {
			for i := range len(queueDataArray) {
				if qe.UUID == queueDataArray[i].UUID {
					queueDataArray = append(queueDataArray[:i], queueDataArray[i+1:]...)

					fetchQueueData()

					break
				}
			}
		}),
	)

	entryPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(qe)
	popUpPos := entryPos.Add(pe.Position)
	c := fyne.CurrentApp().Driver().CanvasForObject(qe)
	widget.ShowPopUpMenuAtPosition(menu, c, popUpPos)
}

func mediaBarConstructor() *fyne.Container {
	// Initialize bottomMediaInfoContainer
	mediaProgressBar := widget.NewSlider(0, 1)
	mediaProgressBar.SetValue(0)
	mediaProgressBar.Disable()

	// placeholder values
	mediaLengthSeconds := 90
	mediaNowSeconds := 0

	mediaDurationLabel := widget.NewLabel("00:00")
	mediaName := widget.NewLabel("No Media Playing")

	// placeholder code for media timer
	go func() {
		for {
			time.Sleep(time.Second / 10)
			mediaProgressBar.Max = float64(mediaLengthSeconds)
			if mediaLengthSeconds > mediaNowSeconds {
				mediaNowSeconds++
			} else {
				mediaNowSeconds = 0
			}

			if mediaNowSeconds == 0 || mediaLengthSeconds == 0 {
				mediaProgressBar.SetValue(0)
			} else {
				mediaProgressBar.SetValue(float64(mediaNowSeconds))
			}

			var formattedText string = fmt.Sprintf("%02d:%02d / %02d:%02d", mediaNowSeconds/60, mediaNowSeconds%60, mediaLengthSeconds/60, mediaLengthSeconds%60)

			mediaDurationLabel.SetText(formattedText)
		}
	}()

	bottomMediaInfoContainer := container.NewVBox(container.NewBorder(nil, nil, mediaDurationLabel, mediaName), mediaProgressBar)

	return bottomMediaInfoContainer
}

func uploadFileLogic(reader fyne.URIReadCloser) error {
	defer reader.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create form file field
	formFile, err := writer.CreateFormFile("audioFile", reader.URI().Name())
	if err != nil {
		return fmt.Errorf("error creating form file: %w", err)
	}

	// Copy filedata to form field
	_, err = io.Copy(formFile, reader)
	if err != nil {
		return fmt.Errorf("error copying file data: %w", err)
	}

	// Close writer & finalize multipart form
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("error closing multipart writer: %w", err)
	}

	// Send request to API
	apiURL := serverURL + "/upload"
	req, err := http.NewRequest(http.MethodPost, apiURL, &requestBody)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Handle the response
	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s", string(respBody))
	}

	return nil
}

func headerBarConstructor(w fyne.Window) *fyne.Container {
	// initialize topbar
	serverSelectionDropdown := widget.NewSelect([]string{"Midnight Cookout", "KBot Testing Grounds", "The Groovers"}, func(s string) {
		prop := canvas.NewRectangle(color.Transparent)
		prop.SetMinSize(fyne.NewSize(50, 50))

		a3 := widget.NewActivity()
		d := dialog.NewCustomWithoutButtons("Requesting Server Data...", container.NewStack(prop, a3), w)
		a3.Start()
		d.Show()

		go func() {
			time.Sleep(time.Second * 3)
			a3.Stop()
			d.Hide()
		}()

	})

	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			fmt.Print(err.Error())
			return
		}
		if reader == nil {
			return
		}

		uploadFileLogic(reader)

	}, w)

	uploadFileButton := widget.NewButton("Upload File", func() {
		fileDialog.Resize(fyne.NewSize(w.Content().Size().Width*.8, w.Content().Size().Height*.8))
		fileDialog.Show()
	})

	connectEntry := widget.NewEntry()
	connectEntry.Text = serverURL

	connectDialog := dialog.NewForm(
		"Connect to KBot",
		"Connect",
		"Cancel",
		[]*widget.FormItem{{Text: "IP Address:Port", Widget: connectEntry}},
		func(b bool) {
			if b { // Runs when pressing 'Connect'
				serverURL = connectEntry.Text
				fmt.Println(serverURL)
			} else { // Runs when dismissed
				return
			}
		},
		w)

	connectButton := widget.NewButton("Connect to Server", func() {
		connectDialog.Resize(fyne.NewSize(430, 100))
		connectDialog.Show()
	})

	buttonsBox := container.NewHBox(uploadFileButton, connectButton)

	topMainContainer := container.NewBorder(nil, nil, buttonsBox, serverSelectionDropdown)

	return topMainContainer
}

func queueBarConstructor() *fyne.Container {

	playerControlsBar := widget.NewToolbar()

	queueList := widget.NewList(
		func() int {
			return len(queueDataArray)
		},
		func() fyne.CanvasObject {

			entry := newQueueEntry()

			return entry
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			co.(*queueEntryWidget).UUID = queueDataArray[lii].UUID
			co.(*queueEntryWidget).SetText(queueDataArray[lii].Title)
		})

	queueBar := container.NewBorder(nil, playerControlsBar, nil, nil, queueList)

	return queueBar
}

func contentConstructor(w fyne.Window) *fyne.Container {
	queueDataArray = []queueEntryData{}
	mediaBar := mediaBarConstructor()
	headerBar := headerBarConstructor(w)
	queueBar := queueBarConstructor()

	content := container.NewBorder(
		headerBar,
		mediaBar,
		nil,
		queueBar,
	)

	return content
}

func main() {
	a := app.New()
	w := a.NewWindow("KBot Media Player " + version)

	content := contentConstructor(w)

	w.SetContent(content)
	w.Resize(fyne.Size{Width: 1024, Height: 768})
	w.ShowAndRun()
}
