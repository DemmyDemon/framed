package server

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	refreshSecondsDay   = 60
	refreshSecondsNight = 1200
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

type Server struct {
	Port      int
	Verbosity int
	Changed   time.Time
	Filename  string
}

func Begin(port int, filename string) error {

	srv := Server{
		Port:     port,
		Filename: filename,
	}

	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("accessing display file: %w", err)
	}
	srv.Changed = info.ModTime()

	return http.ListenAndServe(fmt.Sprintf(":%d", port), &srv)
}

func (srv Server) log(line ...string) {
	log.Println(line)
}
func (srv Server) verbose(level int, line ...string) {
	if srv.Verbosity >= level {
		srv.log(line...)
	}
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	remote := strings.SplitN(r.RemoteAddr, ":", 2)[0]
	req := fmt.Sprintf("[%s] %s %s", remote, r.Method, r.RequestURI)
	srv.log(req)

	if !strings.HasPrefix(remote, "192.168.") {
		srv.log("Request from %s denied outright.\n", r.RemoteAddr)
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
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexhtml)
	case "/image":
		raw, err := os.ReadFile(srv.Filename)

		lines := strings.Split(string(raw), "\n")

		if len(lines) > 15 {
			lines = lines[len(lines)-15:]
		}

		screen := CreateScreen(lines)

		w.Header().Set("Content-Type", "image/png")

		// Stuffing it in a buffer first, because .Encode doesn't report size.
		var buf bytes.Buffer
		err = png.Encode(&buf, screen)
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

		info, err := os.Stat(srv.Filename)
		if err != nil {
			srv.log(fmt.Sprintf("%s failed to poll the file: %s", req, err))
		}
		srv.Changed = info.ModTime()

		refreshrate := refreshSecondsDay
		if time.Now().Hour() < 6 { //   Between midnight and 06:00
			refreshrate = refreshSecondsNight
		}

		resp, err := json.Marshal(DisplayResponse{
			FileName:        fmt.Sprintf("screen-%d.png", srv.Changed.Unix()),
			ImageUrl:        fmt.Sprintf("http://%s/image", r.Host),
			RefreshRate:     refreshrate,
			SpecialFunction: "", // Was "sleep", not sure what it does. Cargo cult.
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
