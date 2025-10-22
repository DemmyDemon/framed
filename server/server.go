package server

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DemmyDemon/framed/ui"
)

const (
	battMaxVolt       = 4.05
	battMinVolt       = 0.45
	battMinPercentage = 10
	battMaxPercentage = 95
)

var batteryVoltage float32

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
	chLog     chan ui.LogEntry
	chText    chan string
}

var muLines sync.Mutex
var Lines = []string{"TODO: Make it load the previous text", "from disk."}

func Begin(port int, verbosity int, chLog chan ui.LogEntry, chText chan string) error {

	srv := Server{
		Port:      port,
		Verbosity: verbosity,
		chLog:     chLog,
		chText:    chText,
	}

	go srv.updateLines()

	return http.ListenAndServe(fmt.Sprintf(":%d", port), srv)
}

func (srv Server) updateLines() {
	for text := range srv.chText {
		srv.verbose(2, "Recieved text")
		muLines.Lock()
		Lines = strings.Split(text, "\n")
		muLines.Unlock()
	}
}

func (srv Server) log(line string) {
	if srv.Verbosity > 0 {
		srv.chLog <- ui.LogEntry{Payload: line}
	}
}
func (srv Server) verbose(level int, line string) {
	if srv.Verbosity >= level {
		srv.chLog <- ui.LogEntry{Payload: line}
	}
}

func (srv Server) battPercent() string {
	if batteryVoltage <= battMinVolt {
		return " !! BATTERY LOW !! "
	}
	if batteryVoltage >= battMaxVolt {
		return ", Battery full"
	}
	percentage := (batteryVoltage-battMinVolt)/(battMaxVolt-battMinVolt)*(battMaxPercentage-battMinPercentage) + battMinPercentage
	return fmt.Sprintf(", Battery %.0f%%", percentage)
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

		battery := srv.battPercent()
		now := time.Now()
		_, week := now.ISOWeek()

		lines := []string{
			fmt.Sprintf("%s, week %d%s", now.Format("15:04 Monday"), week, battery),
		}
		muLines.Lock()
		lines = append(lines, Lines...)
		muLines.Unlock()
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

		voltStr := r.Header.Get("Battery-Voltage")
		if voltStr != "" {
			srv.verbose(2, fmt.Sprintf("[%s] Battery voltage is %s", req, voltStr))
			voltage, err := strconv.ParseFloat(voltStr, 32)
			if err != nil {
				srv.log(err.Error())
			} else {
				batteryVoltage = float32(voltage)
			}
		}

		now := time.Now()
		resp, err := json.Marshal(DisplayResponse{
			FileName:        fmt.Sprintf("screen-%d.png", now.Unix()),
			ImageUrl:        fmt.Sprintf("http://%s/image", r.Host),
			RefreshRate:     max(10, 60-now.Second()),
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
		w.Write([]byte(`Quiet, please`))
		if r.Method != "POST" {
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			srv.log(fmt.Sprintf("Error reading log POST: %s", err))
			return
		}
		srv.verbose(2, string(body))
	}
}
