package server

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DemmyDemon/framed/ui"
)

const (
	battMaxVolt       = 4.05
	battMinVolt       = 0.45
	battMinPercentage = 10
	battMaxPercentage = 95
)

//go:embed html/index.html
var indexhtml []byte

type DisplayResponse struct {
	Status          int    `json:"status"`
	ImageUrl        string `json:"image_url"`
	FileName        string `json:"filename"`
	RefreshRate     int    `json:"refresh_rate"`
	ResetFormware   bool   `json:"reset_firmware"`
	UpdateFirmware  bool   `json:"update_firmware"`
	FirmwareUrl     string `json:"firmware_url"`
	SpecialFunction string `json:"special_function"`
}

type DisplayText struct {
	Time  time.Time
	Lines []string
}

func NewDisplayText(text string) DisplayText {
	return DisplayText{
		Lines: strings.Split(text, "\n"),
		Time:  time.Now(),
	}
}

func (dt *DisplayText) Update(text string) {
	dt.Lines = strings.Split(text, "\n")
	dt.Time = time.Now()
}

type Server struct {
	Port      int
	Verbosity int
	chLog     chan ui.LogEntry
	chText    chan string
	text      DisplayText
}

func Begin(port int, verbosity int, chLog chan ui.LogEntry, chText chan string) error {

	srv := Server{
		Port:      port,
		Verbosity: verbosity,
		chLog:     chLog,
		chText:    chText,
		text:      NewDisplayText("TODO: Make it load the previous text\nfrom disk."),
	}

	go srv.updateLines()

	return http.ListenAndServe(fmt.Sprintf(":%d", port), &srv)
}

func (srv *Server) updateLines() {
	for text := range srv.chText {
		srv.log("Server recieved text update")
		srv.text.Update(text)
	}
}

func (srv Server) log(line ...string) {
	if srv.Verbosity >= 0 {
		srv.chLog <- ui.LogEntry{Payload: line}
	}
}
func (srv Server) verbose(level int, line ...string) {
	if srv.Verbosity >= level {
		srv.chLog <- ui.LogEntry{Payload: line}
	}
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	remote := strings.SplitN(r.RemoteAddr, ":", 2)[0]
	req := fmt.Sprintf("[%s] %s %s", remote, r.Method, r.RequestURI)
	srv.log(req)

	if !strings.HasPrefix(remote, "192.168.") {
		fmt.Printf("Request from %s denied outright.\n", r.RemoteAddr)
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(`Not allowed!`))
		return
	}

	for key, value := range r.Header {
		srv.verbose(2, fmt.Sprintf("    %s %v", key, value))
	}
	switch r.RequestURI {
	case "/":
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/html")
		w.Write(indexhtml)
	case "/image":

		_, week := srv.text.Time.ISOWeek()

		lines := []string{
			fmt.Sprintf("%s, week %d", srv.text.Time.Format("Monday"), week),
		}
		// FIXME: This is probably subject to race conditions
		// There might be situations where it's reading the lines while they are being updated.
		lines = append(lines, srv.text.Lines...)
		screen := CreateScreen(lines)

		w.Header().Set("Content-Type", "image/png")

		// Stuffing it in a buffer first, because .Encode doesn't report size.
		var buf bytes.Buffer
		err := png.Encode(&buf, screen)
		if err != nil {
			srv.log(fmt.Sprintf("[%s] Failed buffer image data: %s", remote, err))
			return
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

		size, err := io.Copy(w, &buf)
		if err != nil {
			srv.log(fmt.Sprintf("[%s] Failed write image data: %s", remote, err))
			return
		}

		srv.log(fmt.Sprintf("[%s] %d bytes of image data sent", remote, size))
		return
	case "/api/display":

		/*
			Note to future self:
			The srv.textTime.Unix() in the "filename" makes the filename only change
			when the text does.
			That means the TRMNL doesn't fetch the image if it hasn't changed.
			This sacrifices "updated" battery display, but whatever. Scrapped feature.
			REMEMBER TO LOWER THE REFRESH RATE IF YOU PUT THIS BACK! XD
		*/

		resp, err := json.Marshal(DisplayResponse{
			FileName:        fmt.Sprintf("screen-%d.png", srv.text.Time.Unix()),
			ImageUrl:        fmt.Sprintf("http://%s/image", r.Host),
			RefreshRate:     10,
			SpecialFunction: "sleep",
		})
		if err != nil {
			srv.log(fmt.Sprintf("%s Failed to give a viable DISPLAY response: %s", req, err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(`Server blew up`))
			return
		}
		srv.verbose(2, fmt.Sprintf("Serving display data: %s", resp))
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	case "/api/log":
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(`Okay, thanks.`))
		if r.Method != "POST" {
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			srv.log(fmt.Sprintf("Error reading log POST: %s", err))
			return
		}
		srv.log(string(body))
	}
}
