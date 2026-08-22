package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Ribco/dinex/agent/internal/config"
	"github.com/Ribco/dinex/agent/internal/manager"
)

type Server struct {
	cfg *config.Config
	mgr *manager.Manager
}

func New(cfg *config.Config, mgr *manager.Manager) *Server {
	return &Server{
		cfg: cfg,
		mgr: mgr,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/api/v1/status", s.status)
	mux.HandleFunc("/api/v1/servers", s.servers)
	mux.HandleFunc("/api/v1/servers/", s.server)

	return s.auth(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "dinex-agent",
		"version": s.cfg.Version,
	})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"node_id":   s.cfg.NodeID,
		"node_name": s.cfg.NodeName,
		"version":   s.cfg.Version,
	})
}

func (s *Server) servers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.mgr.List())

	case http.MethodPost:
		var server manager.Server

		if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if err := s.mgr.Create(&server); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusCreated, server)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) server(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(
		r.URL.Path,
		"/api/v1/servers/",
	)

	parts := strings.Split(
		strings.Trim(path, "/"),
		"/",
	)

	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	id := parts[0]

	if len(parts) == 1 && r.Method == http.MethodGet {
		status, err := s.mgr.Status(id)

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		writeJSON(w, http.StatusOK, status)
		return
	}

	if len(parts) == 2 && parts[1] == "logs" && r.Method == http.MethodGet {
		lines, err := s.mgr.Logs(id)

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"lines": lines,
		})

		return
	}

	if len(parts) == 2 && parts[1] == "install-packages" && r.Method == http.MethodPost {
		var payload struct {
			Packages []string `json:"packages"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		output, err := s.mgr.InstallNodePackages(id, payload.Packages)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":  err.Error(),
				"output": string(output),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"output": string(output),
		})
		return
	}

	if len(parts) == 2 && r.Method == http.MethodPost {
		var err error

		switch parts[1] {
		case "start":
			err = s.mgr.Start(id)

		case "stop":
			err = s.mgr.Stop(id)

		case "kill":
			err = s.mgr.Kill(id)

		default:
			http.NotFound(w, r)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})

		return
	}

	// Files API
	if len(parts) == 2 && parts[1] == "files" && r.Method == http.MethodGet {
		path := r.URL.Query().Get("path")

		files, err := s.mgr.Files(id, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"path":  path,
			"files": files,
		})
		return
	}

	if len(parts) == 2 && parts[1] == "settings" && r.Method == http.MethodPut {
		var payload struct {
			Name      string `json:"name"`
			Startup   string `json:"startup"`
			Directory string `json:"directory"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if err := s.mgr.UpdateServer(
			id,
			payload.Name,
			payload.Startup,
			payload.Directory,
		); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
		return
	}

	if len(parts) == 2 && parts[1] == "file" && r.Method == http.MethodGet {
		path := r.URL.Query().Get("path")

		data, err := s.mgr.ReadFile(id, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(data)
		return
	}

	if len(parts) == 2 && parts[1] == "file" && r.Method == http.MethodPut {
		path := r.URL.Query().Get("path")

		data, err := io.ReadAll(
			http.MaxBytesReader(w, r.Body, 50<<20),
		)
		if err != nil {
			http.Error(w, "failed to read file", http.StatusBadRequest)
			return
		}

		if err := s.mgr.WriteFile(id, path, data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
		return
	}

	if len(parts) == 2 && parts[1] == "folder" && r.Method == http.MethodPost {
		path := r.URL.Query().Get("path")

		if err := s.mgr.CreateFolder(id, path); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{
			"status": "ok",
		})
		return
	}

	if len(parts) == 2 && parts[1] == "files" && r.Method == http.MethodDelete {
		path := r.URL.Query().Get("path")

		if err := s.mgr.DeleteFile(id, path); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
		return
	}

	if len(parts) == 2 && parts[1] == "rename" && r.Method == http.MethodPost {
		var payload struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if err := s.mgr.RenameFile(
			id,
			payload.OldPath,
			payload.NewPath,
		); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
		return
	}

	if len(parts) == 2 && parts[1] == "download" && r.Method == http.MethodGet {
		path := r.URL.Query().Get("path")

		filePath, err := s.mgr.FilePath(id, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		info, err := s.mgr.FileInfo(id, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if info.IsDir() {
			http.Error(w, "path is a directory", http.StatusBadRequest)
			return
		}

		http.ServeFile(w, r, filePath)
		return
	}

	http.NotFound(w, r)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		if s.cfg.AuthToken == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error":   401,
				"message": "The required headers for Agent wasn't found on this request.",
			})
			return
		}

		token := strings.TrimPrefix(
			r.Header.Get("Authorization"),
			"Bearer ",
		)

		if token == "" || token != s.cfg.AuthToken {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error":   401,
				"message": "The required headers for Agent weren't found on this request.",
			})

			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}
