package nodes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Ribco/dinex/panel/internal/database"
)

type Node struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Token     string `json:"-"`
	CreatedAt int64  `json:"created_at"`
}

type Status struct {
	NodeID   int64  `json:"node_id"`
	NodeName string `json:"node_name"`
	Version  string `json:"version"`
	Online   bool   `json:"online"`
	Error    string `json:"error,omitempty"`
}

type Manager struct {
	DB *database.DB
}

func New(db *database.DB) *Manager {
	return &Manager{DB: db}
}

func (m *Manager) Create(name, address, token string) error {
	name = strings.TrimSpace(name)
	address = strings.TrimRight(strings.TrimSpace(address), "/")
	token = strings.TrimSpace(token)

	if name == "" {
		return fmt.Errorf("node name is required")
	}

	if address == "" {
		return fmt.Errorf("node address is required")
	}

	if token == "" {
		return fmt.Errorf("agent token is required")
	}

	_, err := m.DB.SQL.Exec(`
		INSERT INTO nodes (name, address, token, created_at)
		VALUES (?, ?, ?, ?)
	`, name, address, token, database.Now())

	return err
}

func (m *Manager) List() ([]Node, error) {
	rows, err := m.DB.SQL.Query(`
		SELECT id, name, address, token, created_at
		FROM nodes
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Node

	for rows.Next() {
		var n Node

		if err := rows.Scan(
			&n.ID,
			&n.Name,
			&n.Address,
			&n.Token,
			&n.CreatedAt,
		); err != nil {
			return nil, err
		}

		result = append(result, n)
	}

	return result, rows.Err()
}

func (m *Manager) Get(id int64) (*Node, error) {
	var n Node

	err := m.DB.SQL.QueryRow(`
		SELECT id, name, address, token, created_at
		FROM nodes
		WHERE id = ?
	`, id).Scan(
		&n.ID,
		&n.Name,
		&n.Address,
		&n.Token,
		&n.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &n, nil
}

func (m *Manager) Delete(id int64) error {
	_, err := m.DB.SQL.Exec(
		"DELETE FROM nodes WHERE id = ?",
		id,
	)

	return err
}

func (m *Manager) Check(n *Node) Status {
	status := Status{
		NodeID:   n.ID,
		NodeName: n.Name,
	}

	url := strings.TrimRight(n.Address, "/") + "/api/v1/status"

	req, err := http.NewRequest(
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		status.Error = err.Error()
		return status
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+n.Token,
	)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		status.Error = err.Error()
		return status
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		status.Error = fmt.Sprintf(
			"agent returned HTTP %d",
			resp.StatusCode,
		)
		return status
	}

	var data struct {
		NodeID   string `json:"node_id"`
		NodeName string `json:"node_name"`
		Version  string `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		status.Error = err.Error()
		return status
	}

	status.Online = true
	status.NodeName = data.NodeName
	status.Version = data.Version

	return status
}
