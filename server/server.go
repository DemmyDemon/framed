package server

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"image/png"
	"io"
	"net/http"
	"strings"
	"time"
)

//go:embed html/index.html
var pageTemplate string

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
	Template  *template.Template
}

var Lines = []string{"TODO: Make it load the previous text from disk."}

func Begin(port int, verbosity int) error {

	tpl, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		return fmt.Errorf("embedded template: %w", err)
	}

	srv := Server{
		Port:      port,
		Verbosity: verbosity,
		Template:  tpl,
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
		if r.Method == "POST" {
			err := r.ParseForm()
			if err != nil {
				srv.log(fmt.Sprintf("[%s] Failed reading POST: %s", remote, err))
				return
			}
			text := r.PostForm.Get("text")
			Lines = strings.Split(text, "\n")
			for _, line := range Lines {
				srv.verbose(2, line)
			}
		}
		err := srv.Template.Execute(w, struct{ Text string }{Text: strings.Join(Lines, "\n")})
		if err != nil {
			srv.log(fmt.Sprintf("[%s] Failed executing template: %s", remote, err))
		}
	case "/image":
		now := time.Now()
		_, week := now.ISOWeek()
		lines := []string{
			fmt.Sprintf("%s, week %d", now.Format("15:04 Monday"), week),
		}

		lines = append(lines, Lines...)
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
