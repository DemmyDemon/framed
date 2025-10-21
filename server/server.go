package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sergeymakinen/go-bmp"
)

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
}

func Begin(port int, verbosity int) error {
	srv := Server{
		Port:      port,
		Verbosity: verbosity,
	}
	return http.ListenAndServe(fmt.Sprintf(":%d", port), srv)
}

func (srv Server) log(line string) {
	if srv.Verbosity > 0 {
		fmt.Println(line)
	}
}
func (srv Server) verbose(level int, line string) {
	if srv.Verbosity >= level {
		srv.log(line)
	}
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	remote := strings.SplitN(r.RemoteAddr, ":", 2)[0]
	req := fmt.Sprintf("[%s] %s %s", remote, r.Method, r.RequestURI)
	srv.log(req)

	if !strings.HasPrefix(remote, "192.168.") {
		fmt.Printf("Request from %s denied outright.\n", r.RemoteAddr)
		w.WriteHeader(http.StatusForbidden)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte(`Not allowed!`))
		return
	}

	for key, value := range r.Header {
		srv.verbose(2, fmt.Sprintf("    %s %v", key, value))
	}
	switch r.RequestURI {
	case "/image":
		screen := CreateScreen([]string{
			"This is placeholder text",
			"This is *more* placeholder text",
		})
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "image/bmp")

		var buf bytes.Buffer
		err := bmp.Encode(&buf, screen)
		if err != nil {
			srv.log(fmt.Sprintf("[%s] Failed buffer image data: %s", remote, err))
			return
		}

		size, err := io.Copy(w, &buf)
		if err != nil {
			srv.log(fmt.Sprintf("[%s] Failed write image data: %s", remote, err))
			return
		}

		srv.log(fmt.Sprintf("[%s] %d bytes of image data sent", remote, size))
		return
	case "/api/display":
		resp, err := json.Marshal(DisplayResponse{
			FileName:        fmt.Sprintf("screen-%d.bmp", time.Now().Unix()/10),
			ImageUrl:        fmt.Sprintf("http://%s/image", r.Host),
			RefreshRate:     60,
			SpecialFunction: "sleep",
		})
		if err != nil {
			srv.log(fmt.Sprintf("%s Failed to give a viable DISPLAY response: %s", req, err))
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Add("Content-Type", "text/plain")
			w.Write([]byte(`Server blew up`))
			return
		}
		srv.verbose(2, fmt.Sprintf("Serving display data: %s", resp))
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")
		w.Write(resp)
	case "/api/log":
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
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
