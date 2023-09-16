package tilehandler

import (
	"image/png"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/simonhege/varmomapo/pkg/drawers"
)

var urlRegex = regexp.MustCompile(`\A/.*/(\w+)/(\d+)/(\d+)/(\d+)\.(png|jpeg)\z`)

func New(drawerFactory func(layerName string) drawers.Drawer) http.Handler {
	return &server{drawerFactory: drawerFactory}
}

type server struct {
	drawerFactory func(layerName string) drawers.Drawer
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("request received", "host", r.URL.Host, "path", r.URL.Path)

	m := urlRegex.FindStringSubmatch(r.URL.Path)
	if m == nil || len(m) != 6 {
		slog.WarnContext(r.Context(), "Invalid URL", "path", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	//layer name
	layerName := m[1]
	drawer := s.drawerFactory(layerName)
	if drawer == nil {
		slog.WarnContext(r.Context(), "layer not found", m[1])
		http.NotFound(w, r)
		return
	}

	// Decode level, x and y
	level, err := strconv.Atoi(m[2])
	if err != nil {
		slog.WarnContext(r.Context(), "level decoding failed", m[2])
		http.Error(w, "level decoding failed", http.StatusBadRequest)
		return
	}
	x, err := strconv.Atoi(m[3])
	if err != nil {
		slog.WarnContext(r.Context(), "x decoding failed", m[3])
		http.Error(w, "x decoding failed", http.StatusBadRequest)
		return
	}
	y, err := strconv.Atoi(m[4])
	if err != nil {
		slog.WarnContext(r.Context(), "y decoding failed", m[4])
		http.Error(w, "y decoding failed", http.StatusBadRequest)
		return
	}

	img, err := drawer.Draw(r.Context(), x, y, level)
	if err != nil {
		slog.Error("tile generation failed", "error", err)
		http.Error(w, "tile generation failed", http.StatusInternalServerError)
		return
	}

	if err := png.Encode(w, img); err != nil {
		slog.Error("tile encoding failed", "error", err)
		http.Error(w, "tile encoding failed", http.StatusInternalServerError)
		return
	}

	slog.Info("request completed")
}
