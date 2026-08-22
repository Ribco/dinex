package api

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ribco/dinex/panel/internal/auth"
	"github.com/Ribco/dinex/panel/internal/nodes"
	"github.com/Ribco/dinex/panel/internal/servers"
)

type ServerAPI struct {
	Servers *servers.Manager
	Nodes   *nodes.Manager
	Auth    *auth.Auth
}

func NewServerAPI(manager *servers.Manager, nodeManager *nodes.Manager, authManager *auth.Auth) *ServerAPI {
	return &ServerAPI{Servers: manager, Nodes: nodeManager, Auth: authManager}
}

func (a *ServerAPI) Page(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		list, err := a.Servers.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type ServerView struct {
			Server servers.Server
			Status servers.Status
			Error  string
		}

		views := make([]ServerView, 0, len(list))

		for _, server := range list {
			status, err := a.Servers.Status(server)

			view := ServerView{
				Server: server,
				Status: status,
			}

			if err != nil {
				view.Error = err.Error()
			}

			views = append(views, view)
		}

		tmpl, err := template.ParseFiles(
			"panel/web/templates/servers.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nodeList, err := a.Nodes.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Servers []ServerView
			Nodes   []nodes.Node
		}{Servers: views, Nodes: nodeList}
		_ = tmpl.Execute(w, data)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		nodeID, err := strconv.ParseInt(r.FormValue("node_id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid node ID", http.StatusBadRequest)
			return
		}

		userID, ok := a.Auth.UserFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"401": "The required headers for Panel were not found on this request."})
			return
		}

		env := map[string]string{}

		node, err := a.Nodes.Get(nodeID)
		if err != nil {
			http.Error(w, "node not found", http.StatusBadRequest)
			return
		}
		if !a.Nodes.Check(node).Online {
			http.Error(w, "Panel cannot communicate with Agent. Contact Node Maintainer.", http.StatusBadGateway)
			return
		}

		if err := a.Servers.Create(
			userID,
			nodeID,
			r.FormValue("id"),
			r.FormValue("name"),
			r.FormValue("startup"),
			env,
		); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/servers", http.StatusSeeOther)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (a *ServerAPI) Action(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/servers/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		http.Error(w, "invalid server action", http.StatusBadRequest)
		return
	}

	server, err := a.Servers.GetByUUID(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	action := parts[1]
	if action == "restart" {
		if err := a.Servers.Action(*server, "stop"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		action = "start"
	}

	if err := a.Servers.Action(*server, action); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/servers/"+server.UUID, http.StatusSeeOther)
}

func (a *ServerAPI) ServerPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/servers/"), "/")
	if uuid == "" || strings.Contains(uuid, "/") {
		http.NotFound(w, r)
		return
	}

	server, err := a.Servers.GetByUUID(uuid)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	status, statusErr := a.Servers.Status(*server)

	data := struct {
		Server servers.Server
		Status servers.Status
		Error  string
	}{
		Server: *server,
		Status: status,
	}

	if statusErr != nil {
		data.Error = statusErr.Error()
	}

	tmpl, err := template.ParseFiles("panel/web/templates/server.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = tmpl.Execute(w, data)
}

func (a *ServerAPI) SettingsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid := strings.TrimSuffix(
		strings.TrimPrefix(r.URL.Path, "/api/servers/"),
		"/settings",
	)

	server, err := a.Servers.GetByUUID(uuid)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var payload struct {
		Name        string `json:"name"`
		ExternalID  string `json:"external_id"`
		Description string `json:"description"`
		Startup     string `json:"startup"`
		Directory   string `json:"directory"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := a.Servers.UpdateSettings(
		*server,
		payload.Name,
		payload.ExternalID,
		payload.Description,
		payload.Startup,
		payload.Directory,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (a *ServerAPI) LogsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uuid := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/servers/"), "/logs")
	server, err := a.Servers.GetByUUID(uuid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lines, err := a.Servers.Logs(*server)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"lines": lines})
}

func (a *ServerAPI) FilesAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uuid := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/servers/"), "/files")
	server, err := a.Servers.GetByUUID(uuid)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data, status, contentType, err := a.Servers.Files(*server, r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func (a *ServerAPI) FileAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/servers/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	uuid := parts[0]
	action := parts[1]

	server, err := a.Servers.GetByUUID(uuid)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "file":
		if r.Method == http.MethodGet {
			data, status, contentType, err := a.Servers.ReadFile(
				*server,
				r.URL.Query().Get("path"),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(status)
			_, _ = w.Write(data)
			return
		}

		if r.Method == http.MethodPut {
			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 50<<20))
			if err != nil {
				http.Error(w, "failed to read file", http.StatusBadRequest)
				return
			}

			status, err := a.Servers.WriteFile(
				*server,
				r.URL.Query().Get("path"),
				data,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			w.WriteHeader(status)
			return
		}

	case "folder":
		if r.Method == http.MethodPost {
			status, err := a.Servers.CreateFolder(
				*server,
				r.URL.Query().Get("path"),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			w.WriteHeader(status)
			return
		}

	case "files":
		if r.Method == http.MethodGet {
			data, status, contentType, err := a.Servers.Files(
				*server,
				r.URL.Query().Get("path"),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(status)
			_, _ = w.Write(data)
			return
		}

		if r.Method == http.MethodDelete {
			status, err := a.Servers.DeleteFile(
				*server,
				r.URL.Query().Get("path"),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			w.WriteHeader(status)
			return
		}

	case "rename":
		if r.Method == http.MethodPost {
			data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024*1024))
			if err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			var payload struct {
				OldPath string `json:"old_path"`
				NewPath string `json:"new_path"`
			}

			if err := json.Unmarshal(data, &payload); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}

			status, err := a.Servers.RenameFile(
				*server,
				payload.OldPath,
				payload.NewPath,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			w.WriteHeader(status)
			return
		}

	case "download":
		if r.Method == http.MethodGet {
			data, status, contentType, err := a.Servers.DownloadFile(
				*server,
				r.URL.Query().Get("path"),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}

			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(status)
			_, _ = w.Write(data)
			return
		}
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (a *ServerAPI) JSON(w http.ResponseWriter, r *http.Request) {
	list, err := a.Servers.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Response struct {
		Server servers.Server `json:"server"`
		Status servers.Status `json:"status"`
		Error  string         `json:"error,omitempty"`
	}

	result := make([]Response, 0, len(list))

	for _, server := range list {
		status, err := a.Servers.Status(server)

		item := Response{
			Server: server,
			Status: status,
		}

		if err != nil {
			item.Error = err.Error()
		}

		result = append(result, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
