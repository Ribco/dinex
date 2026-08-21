package manager

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Ribco/dinex/agent/internal/config"
)

type process struct {
	cmd       *exec.Cmd
	startedAt time.Time
	state     string
	exitCode  *int
}

type Manager struct {
	cfg       *config.Config
	mu        sync.RWMutex
	servers   map[string]*Server
	processes map[string]*process
	logs      map[string][]string
	logMu     sync.RWMutex
}

func New(cfg *config.Config) *Manager {
	m := &Manager{
		cfg:       cfg,
		servers:   make(map[string]*Server),
		processes: make(map[string]*process),
		logs:      make(map[string][]string),
	}

	serversDir := filepath.Join(cfg.DataDir, "servers")

	if err := os.MkdirAll(serversDir, 0755); err != nil {
		return m
	}

	if err := m.loadServers(); err != nil {
		// Don't prevent the Agent from starting because one server
		// configuration is broken.
	}

	return m
}

func (m *Manager) loadServers() error {
	root := filepath.Join(m.cfg.DataDir, "servers")

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		serverFile := filepath.Join(
			root,
			entry.Name(),
			"server.json",
		)

		data, err := os.ReadFile(serverFile)
		if err != nil {
			continue
		}

		var server Server

		if err := json.Unmarshal(data, &server); err != nil {
			continue
		}

		if server.ID == "" {
			continue
		}

		if server.Directory == "" {
			server.Directory = filepath.Join(
				root,
				server.ID,
			)
		}

		if server.Env == nil {
			server.Env = make(map[string]string)
		}

		m.servers[server.ID] = &server
	}

	return nil
}

func (m *Manager) saveServer(server *Server) error {
	directory := filepath.Join(
		m.cfg.DataDir,
		"servers",
		server.ID,
	)

	if err := os.MkdirAll(directory, 0750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(
		server,
		"",
		"  ",
	)

	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(directory, "server.json"),
		append(data, '\n'),
		0600,
	)
}

func (m *Manager) Create(s *Server) error {
	if s.ID == "" {
		return errors.New("server ID is required")
	}

	if s.Name == "" {
		return errors.New("server name is required")
	}

	if s.Startup == "" {
		return errors.New("startup command is required")
	}

	if strings.ContainsAny(s.ID, `/\`) ||
		s.ID == "." ||
		s.ID == ".." {
		return errors.New("invalid server ID")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[s.ID]; exists {
		return errors.New("server already exists")
	}

	if s.Directory == "" {
		s.Directory = filepath.Join(
			m.cfg.DataDir,
			"servers",
			s.ID,
		)
	}

	if err := os.MkdirAll(s.Directory, 0750); err != nil {
		return err
	}

	if s.Env == nil {
		s.Env = make(map[string]string)
	}

	s.CreatedAt = time.Now().UTC()

	if err := m.saveServer(s); err != nil {
		return err
	}

	m.servers[s.ID] = s

	return nil
}

func (m *Manager) List() []*Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Server, 0, len(m.servers))

	for _, server := range m.servers {
		copy := *server
		result = append(result, &copy)
	}

	return result
}

func (m *Manager) Get(id string) (*Server, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[id]

	return server, ok
}

type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
}

func (m *Manager) serverPath(id, requested string) (string, error) {
	server, ok := m.Get(id)
	if !ok {
		return "", errors.New("server not found")
	}

	root, err := filepath.Abs(server.Directory)
	if err != nil {
		return "", err
	}

	requested = strings.TrimSpace(requested)
	requested = strings.TrimPrefix(requested, "/")

	target := filepath.Join(root, filepath.FromSlash(requested))

	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("invalid file path")
	}

	return abs, nil
}

func (m *Manager) Files(id, requested string) ([]FileEntry, error) {
	dir, err := m.serverPath(id, requested)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, errors.New("not a directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := make([]FileEntry, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		relative, err := filepath.Rel(
			filepath.Dir(dir),
			filepath.Join(dir, entry.Name()),
		)
		if err != nil {
			continue
		}

		result = append(result, FileEntry{
			Name:    entry.Name(),
			Path:    filepath.ToSlash(filepath.Join(requested, entry.Name())),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime().UTC().Format(time.RFC3339),
		})

		_ = relative
	}

	return result, nil
}

func (m *Manager) ReadFile(id, requested string) ([]byte, error) {
	path, err := m.serverPath(id, requested)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, errors.New("path is a directory")
	}

	return os.ReadFile(path)
}

func (m *Manager) WriteFile(id, requested string, data []byte) error {
	path, err := m.serverPath(id, requested)
	if err != nil {
		return err
	}

	parent := filepath.Dir(path)

	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() {
		return errors.New("parent directory does not exist")
	}

	return os.WriteFile(path, data, 0640)
}

func (m *Manager) CreateFolder(id, requested string) error {
	path, err := m.serverPath(id, requested)
	if err != nil {
		return err
	}

	if err := os.Mkdir(path, 0750); err != nil {
		return err
	}

	return nil
}

func (m *Manager) DeleteFile(id, requested string) error {
	path, err := m.serverPath(id, requested)
	if err != nil {
		return err
	}

	if path == filepath.Clean(m.serverPathRoot(id)) {
		return errors.New("cannot delete server root")
	}

	return os.RemoveAll(path)
}

func (m *Manager) RenameFile(id, oldPath, newPath string) error {
	oldAbs, err := m.serverPath(id, oldPath)
	if err != nil {
		return err
	}

	newAbs, err := m.serverPath(id, newPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(oldAbs); err != nil {
		return err
	}

	if _, err := os.Stat(newAbs); err == nil {
		return errors.New("destination already exists")
	}

	return os.Rename(oldAbs, newAbs)
}

func (m *Manager) serverPathRoot(id string) string {
	server, ok := m.Get(id)
	if !ok {
		return ""
	}

	root, _ := filepath.Abs(server.Directory)
	return root
}

func (m *Manager) FileInfo(id, requested string) (os.FileInfo, error) {
	path, err := m.serverPath(id, requested)
	if err != nil {
		return nil, err
	}

	return os.Stat(path)
}

func (m *Manager) FilePath(id, requested string) (string, error) {
	path, err := m.serverPath(id, requested)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	return path, nil
}

func (m *Manager) UpdateServer(id, name, startup, directory string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[id]
	if !ok {
		return errors.New("server not found")
	}

	if name != "" {
		server.Name = name
	}

	if startup != "" {
		server.Startup = startup
	}

	// Only change the directory when one was explicitly supplied.
	// An empty value keeps the server's existing directory.
	if strings.TrimSpace(directory) != "" {
		server.Directory = directory
	}

	return m.saveServer(server)
}

func (m *Manager) Start(id string) error {
	m.mu.Lock()

	server, exists := m.servers[id]

	if !exists {
		m.mu.Unlock()
		return errors.New("server not found")
	}

	if p, ok := m.processes[id]; ok &&
		p.state == "running" {
		m.mu.Unlock()
		return errors.New("server is already running")
	}

	command := exec.Command(
		"sh",
		"-c",
		server.Startup,
	)

	command.Dir = server.Directory
	command.Env = os.Environ()

	for key, value := range server.Env {
		command.Env = append(
			command.Env,
			key+"="+value,
		)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		m.mu.Unlock()
		return err
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		m.mu.Unlock()
		return err
	}

	if err := command.Start(); err != nil {
		m.mu.Unlock()
		return err
	}

	m.processes[id] = &process{
		cmd:       command,
		startedAt: time.Now().UTC(),
		state:     "running",
	}

	m.mu.Unlock()

	go m.capture(id, stdout)
	go m.capture(id, stderr)
	go m.wait(id, command)

	return nil
}

func (m *Manager) wait(id string, command *exec.Cmd) {
	err := command.Wait()

	exitCode := 0

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.processes[id]; ok {
		p.state = "stopped"
		p.exitCode = &exitCode
	}
}

func (m *Manager) capture(
	id string,
	reader interface {
		Read([]byte) (int, error)
	},
) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		m.logMu.Lock()

		m.logs[id] = append(
			m.logs[id],
			scanner.Text(),
		)

		if len(m.logs[id]) > 2000 {
			m.logs[id] =
				m.logs[id][len(m.logs[id])-2000:]
		}

		m.logMu.Unlock()
	}
}

func (m *Manager) Stop(id string) error {
	m.mu.RLock()

	p, exists := m.processes[id]

	m.mu.RUnlock()

	if !exists ||
		p.cmd == nil ||
		p.cmd.Process == nil ||
		p.state != "running" {
		return errors.New("server is not running")
	}

	return p.cmd.Process.Signal(os.Interrupt)
}

func (m *Manager) Kill(id string) error {
	m.mu.RLock()

	p, exists := m.processes[id]

	m.mu.RUnlock()

	if !exists ||
		p.cmd == nil ||
		p.cmd.Process == nil ||
		p.state != "running" {
		return errors.New("server is not running")
	}

	return p.cmd.Process.Kill()
}

func (m *Manager) Status(id string) (Status, error) {
	server, exists := m.Get(id)

	if !exists {
		return Status{}, errors.New("server not found")
	}

	m.mu.RLock()

	p, running := m.processes[id]

	m.mu.RUnlock()

	result := Status{
		ID:    server.ID,
		Name:  server.Name,
		State: "offline",
	}

	if !running {
		return result, nil
	}

	result.State = p.state

	if p.cmd != nil &&
		p.cmd.Process != nil &&
		p.state == "running" {
		result.PID = p.cmd.Process.Pid
	}

	result.StartedAt = p.startedAt.Format(time.RFC3339)
	result.ExitCode = p.exitCode

	return result, nil
}

func (m *Manager) Logs(id string) ([]string, error) {
	if _, exists := m.Get(id); !exists {
		return nil, errors.New("server not found")
	}

	m.logMu.RLock()
	defer m.logMu.RUnlock()

	return append(
		[]string(nil),
		m.logs[id]...,
	), nil
}
