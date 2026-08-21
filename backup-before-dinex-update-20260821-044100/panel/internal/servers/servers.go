package servers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Ribco/dinex/panel/internal/database"
	"github.com/Ribco/dinex/panel/internal/nodes"
	"github.com/google/uuid"
)

type Server struct {
	ID         int64  `json:"id"`
	UUID       string `json:"uuid"`
	NodeID     int64  `json:"node_id"`
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	CreatedAt  int64  `json:"created_at"`
}

type AgentServer struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Directory string            `json:"directory"`
	Startup   string            `json:"startup"`
	Env       map[string]string `json:"env"`
	CreatedAt time.Time         `json:"created_at"`
}

type Status struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	PID       int    `json:"pid,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

type Manager struct {
	DB    *database.DB
	Nodes *nodes.Manager
}

func New(db *database.DB, nodeManager *nodes.Manager) *Manager {
	return &Manager{
		DB:    db,
		Nodes: nodeManager,
	}
}

func (m *Manager) List() ([]Server, error) {
	rows, err := m.DB.SQL.Query(`
SELECT id, user_id, uuid, node_id, external_id, name, created_at
FROM servers
ORDER BY id DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Server

	for rows.Next() {
		var s Server

		var userID *int64

		if err := rows.Scan(
			&s.ID,
			&userID,
			&s.UUID,
			&s.NodeID,
			&s.ExternalID,
			&s.Name,
			&s.CreatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, s)
	}

	return result, rows.Err()
}

func (m *Manager) GetByUUID(uuid string) (*Server, error) {
	var s Server

	err := m.DB.SQL.QueryRow(`
SELECT id, uuid, node_id, external_id, name, created_at
FROM servers
WHERE uuid = ?
`, uuid).Scan(
		&s.ID,
		&s.UUID,
		&s.NodeID,
		&s.ExternalID,
		&s.Name,
		&s.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (m *Manager) Create(
	userID int64,
	nodeID int64,
	id string,
	name string,
	startup string,
	env map[string]string,
) error {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	startup = strings.TrimSpace(startup)

	serverUUID := uuid.New().String()
	if id == "" {
		return errors.New("server ID is required")
	}

	if name == "" {
		return errors.New("server name is required")
	}

	if startup == "" {
		return errors.New("startup command is required")
	}

	node, err := m.Nodes.Get(nodeID)
	if err != nil {
		return errors.New("node not found")
	}

	payload := AgentServer{
		ID:      id,
		Name:    name,
		Startup: startup,
		Env:     env,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := m.request(
		node,
		http.MethodPost,
		"/api/v1/servers",
		body,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agent returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var created AgentServer

	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return err
	}

	_, err = m.DB.SQL.Exec(`
INSERT INTO servers
(user_id, uuid, node_id, external_id, name, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`,
		userID,
		serverUUID,
		nodeID,
		created.ID,
		created.Name,
		database.Now())

	return err
}

func (m *Manager) Status(server Server) (Status, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return Status{}, errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodGet,
		"/api/v1/servers/"+server.ExternalID,
		nil,
	)
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Status{}, fmt.Errorf("agent returned HTTP %d", resp.StatusCode)
	}

	var status Status

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return Status{}, err
	}

	return status, nil
}

func (m *Manager) Action(server Server, action string) error {
	if action != "start" && action != "stop" && action != "kill" {
		return errors.New("invalid action")
	}

	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodPost,
		"/api/v1/servers/"+server.ExternalID+"/"+action,
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agent returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return nil
}

func (m *Manager) Logs(server Server) ([]string, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return nil, errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodGet,
		"/api/v1/servers/"+server.ExternalID+"/logs",
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("agent returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Lines []string `json:"lines"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Lines, nil
}

func (m *Manager) Files(server Server, path string) ([]byte, int, string, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return nil, 0, "", errors.New("node not found")
	}

	escaped := url.QueryEscape(path)
	resp, err := m.request(
		node,
		http.MethodGet,
		"/api/v1/servers/"+server.ExternalID+"/files?path="+escaped,
		nil,
	)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header.Get("Content-Type"), err
	}

	return data, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

func (m *Manager) ReadFile(server Server, path string) ([]byte, int, string, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return nil, 0, "", errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodGet,
		"/api/v1/servers/"+server.ExternalID+"/file?path="+url.QueryEscape(path),
		nil,
	)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header.Get("Content-Type"), err
	}

	return data, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

func (m *Manager) WriteFile(server Server, path string, body []byte) (int, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return 0, errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodPut,
		"/api/v1/servers/"+server.ExternalID+"/file?path="+url.QueryEscape(path),
		body,
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (m *Manager) CreateFolder(server Server, path string) (int, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return 0, errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodPost,
		"/api/v1/servers/"+server.ExternalID+"/folder?path="+url.QueryEscape(path),
		nil,
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (m *Manager) DeleteFile(server Server, path string) (int, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return 0, errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodDelete,
		"/api/v1/servers/"+server.ExternalID+"/files?path="+url.QueryEscape(path),
		nil,
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (m *Manager) RenameFile(server Server, oldPath, newPath string) (int, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return 0, errors.New("node not found")
	}

	payload := []byte(`{"old_path":` + strconv.Quote(oldPath) +
		`,"new_path":` + strconv.Quote(newPath) + `}`)

	resp, err := m.request(
		node,
		http.MethodPost,
		"/api/v1/servers/"+server.ExternalID+"/rename",
		payload,
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (m *Manager) DownloadFile(server Server, path string) ([]byte, int, string, error) {
	node, err := m.Nodes.Get(server.NodeID)
	if err != nil {
		return nil, 0, "", errors.New("node not found")
	}

	resp, err := m.request(
		node,
		http.MethodGet,
		"/api/v1/servers/"+server.ExternalID+"/download?path="+url.QueryEscape(path),
		nil,
	)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header.Get("Content-Type"), err
	}

	return data, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

func (m *Manager) request(
	node *nodes.Node,
	method string,
	path string,
	body []byte,
) (*http.Response, error) {
	url := strings.TrimRight(node.Address, "/") + path

	var reader *bytes.Reader

	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+node.Token)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return client.Do(req)
}
