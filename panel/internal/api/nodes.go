package api

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ribco/dinex/panel/internal/nodes"
)

type NodeAPI struct {
	Nodes *nodes.Manager
}

func NewNodeAPI(manager *nodes.Manager) *NodeAPI {
	return &NodeAPI{Nodes: manager}
}

func (a *NodeAPI) Page(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.listPage(w)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.FormValue("name")
	address := r.FormValue("address")
	token := r.FormValue("token")

	if err := a.Nodes.Create(name, address, token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}

func (a *NodeAPI) listPage(w http.ResponseWriter) {
	list, err := a.Nodes.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type NodeView struct {
		Node   nodes.Node
		Status nodes.Status
	}

	var views []NodeView

	for _, n := range list {
		views = append(views, NodeView{
			Node:   n,
			Status: a.Nodes.Check(&n),
		})
	}

	tmpl, err := template.ParseFiles(
		"panel/web/templates/nodes.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = tmpl.Execute(w, views)
}

func (a *NodeAPI) JSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	list, err := a.Nodes.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type NodeResponse struct {
		ID        int64        `json:"id"`
		Name      string       `json:"name"`
		Address   string       `json:"address"`
		CreatedAt int64        `json:"created_at"`
		Status    nodes.Status `json:"status"`
	}

	result := make([]NodeResponse, 0, len(list))

	for _, n := range list {
		result = append(result, NodeResponse{
			ID:        n.ID,
			Name:      n.Name,
			Address:   n.Address,
			CreatedAt: n.CreatedAt,
			Status:    a.Nodes.Check(&n),
		})
	}

	_ = json.NewEncoder(w).Encode(result)
}

func (a *NodeAPI) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/nodes/")
	path = strings.TrimSuffix(path, "/delete")

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "invalid node id", http.StatusBadRequest)
		return
	}

	if err := a.Nodes.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/nodes", http.StatusSeeOther)
}
