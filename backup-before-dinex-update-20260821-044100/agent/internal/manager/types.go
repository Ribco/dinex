package manager

import "time"

type Server struct {
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
