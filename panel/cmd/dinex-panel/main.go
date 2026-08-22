package main

import (
	"bufio"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ribco/dinex/panel/internal/api"
	"github.com/Ribco/dinex/panel/internal/auth"
	"github.com/Ribco/dinex/panel/internal/database"
	"github.com/Ribco/dinex/panel/internal/nodes"
	"github.com/Ribco/dinex/panel/internal/servers"
)

const version = "0.1.0"

type App struct {
	DB    *database.DB
	Auth  *auth.Auth
	Nodes *nodes.Manager
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "admin:create":
			adminCreate()
			return

		case "admin:list":
			adminList()
			return

		case "version":
			fmt.Println("Dinex Panel", version)
			return

		case "serve":
			serve()
			return

		case "help":
			help()
			return

		default:
			fmt.Printf("Unknown command: %s\n\n", os.Args[1])
			help()
			os.Exit(1)
		}
	}

	serve()
}

func openDatabase() (*database.DB, *auth.Auth) {
	dataDir := os.Getenv("DINEX_PANEL_DATA")

	if dataDir == "" {
		dataDir = "./data"
	}

	if err := os.MkdirAll(dataDir, 0750); err != nil {
		log.Fatal(err)
	}

	db, err := database.Open(
		filepath.Join(dataDir, "dinex.db"),
	)
	if err != nil {
		log.Fatal(err)
	}

	return db, auth.New(db)
}

func adminCreate() {
	fs := flag.NewFlagSet("admin:create", flag.ExitOnError)

	username := fs.String(
		"username",
		"",
		"admin username",
	)

	password := fs.String(
		"password",
		"",
		"admin password",
	)

	_ = fs.Parse(os.Args[2:])

	reader := bufio.NewReader(os.Stdin)

	if *username == "" {
		fmt.Print("Username: ")

		value, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		*username = strings.TrimSpace(value)
	}

	if *password == "" {
		fmt.Print("Password: ")

		value, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		*password = strings.TrimSpace(value)
	}

	if *username == "" || *password == "" {
		log.Fatal("username and password are required")
	}

	db, authManager := openDatabase()
	defer db.SQL.Close()

	if err := authManager.CreateUser(
		*username,
		*password,
	); err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"✓ Admin user %q created successfully.\n",
		*username,
	)
}

func adminList() {
	db, _ := openDatabase()
	defer db.SQL.Close()

	rows, err := db.SQL.Query(`
		SELECT id, username, created_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	fmt.Println("Dinex administrators:")
	fmt.Println()

	for rows.Next() {
		var (
			id        int64
			username  string
			createdAt int64
		)

		if err := rows.Scan(
			&id,
			&username,
			&createdAt,
		); err != nil {
			log.Fatal(err)
		}

		fmt.Printf(
			"  #%d  %s  (%d)\n",
			id,
			username,
			createdAt,
		)
	}
}

func serve() {
	dataDir := os.Getenv("DINEX_PANEL_DATA")

	if dataDir == "" {
		dataDir = "./data"
	}

	if err := os.MkdirAll(dataDir, 0750); err != nil {
		log.Fatal(err)
	}

	db, err := database.Open(
		filepath.Join(dataDir, "dinex.db"),
	)
	if err != nil {
		log.Fatal(err)
	}

	defer db.SQL.Close()

	app := &App{
		DB:    db,
		Auth:  auth.New(db),
		Nodes: nodes.New(db),
	}

	serverManager := servers.New(db, app.Nodes)
	serverAPI := api.NewServerAPI(serverManager, app.Nodes, app.Auth)

	mux := http.NewServeMux()

	// Shared Dinex static assets.
	mux.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("panel/web/static")),
		),
	)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	mux.HandleFunc("/health", health)
	mux.HandleFunc("/login", app.login)

	nodeAPI := api.NewNodeAPI(app.Nodes)

	mux.Handle(
		"/nodes",
		app.Auth.Require(
			http.HandlerFunc(nodeAPI.Page),
		),
	)

	mux.Handle(
		"/nodes/",
		app.Auth.Require(
			http.HandlerFunc(nodeAPI.Delete),
		),
	)

	mux.Handle(
		"/api/nodes",
		app.Auth.Require(
			http.HandlerFunc(nodeAPI.JSON),
		),
	)

	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DEBUG /servers: method=%s cookie=%q", r.Method, func() string {
			c, err := r.Cookie("dinex_session")
			if err != nil {
				return ""
			}
			return c.Value
		}())
		app.Auth.Require(http.HandlerFunc(serverAPI.Page)).ServeHTTP(w, r)
	})

	mux.Handle(
		"/servers/",
		app.Auth.Require(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					serverAPI.ServerPage(w, r)
					return
				}

				serverAPI.Action(w, r)
			}),
		),
	)

	mux.Handle(
		"/api/servers",
		app.Auth.Require(
			http.HandlerFunc(serverAPI.JSON),
		),
	)

	// Server file manager API.
	mux.Handle(
		"/api/servers/",
		app.Auth.Require(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, "/api/servers/")
				parts := strings.Split(strings.Trim(path, "/"), "/")

				if len(parts) >= 2 && parts[1] == "logs" {
					serverAPI.LogsAPI(w, r)
					return
				}
				if len(parts) >= 2 && parts[1] == "settings" {
					serverAPI.SettingsAPI(w, r)
					return
				}
				if len(parts) >= 2 && (parts[1] == "files" || parts[1] == "rename") {
					serverAPI.FilesAPI(w, r)
					return
				}

				serverAPI.FileAPI(w, r)
			}),
		),
	)

	mux.Handle(
		"/dashboard",
		app.Auth.Require(
			http.HandlerFunc(app.dashboard),
		),
	)

	log.Printf("🦖 Dinex Panel v%s", version)
	log.Println("Listening on 0.0.0.0:8000")

	if err := http.ListenAndServe(
		":8000",
		mux,
	); err != nil {
		log.Fatal(err)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	_, _ = w.Write([]byte(
		`{"status":"ok","service":"dinex-panel","version":"0.1.0"}`,
	))
}

func (a *App) login(
	w http.ResponseWriter,
	r *http.Request,
) {
	tmpl, err := template.ParseFiles(
		"panel/web/templates/login.html",
	)
	if err != nil {
		http.Error(w, "Dinex Panel received an error. There was a problem making this UI. Visit Browser Console for more.", 500)
		return
	}

	if r.Method == http.MethodGet {
		_ = tmpl.Execute(w, nil)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	token, err := a.Auth.Login(
		username,
		password,
	)
	if err != nil {
		_ = tmpl.Execute(w, map[string]string{
			"Error": "Invalid username or password",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "dinex_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   86400,
	})

	http.Redirect(
		w,
		r,
		"/dashboard",
		http.StatusSeeOther,
	)
}

func (a *App) dashboard(
	w http.ResponseWriter,
	r *http.Request,
) {
	tmpl, err := template.ParseFiles(
		"panel/web/templates/dashboard.html",
	)
	if err != nil {
		http.Error(w, "Dinex Panel received an error. There was a problem making this UI. Visit Browser Console for more.", http.StatusInternalServerError)
		return
	}

	serverManager := servers.New(a.DB, a.Nodes)

	list, err := serverManager.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type DashboardServer struct {
		Server servers.Server
		Status servers.Status
		Error  string
	}

	items := make([]DashboardServer, 0, len(list))

	for _, server := range list {
		status, statusErr := serverManager.Status(server)

		item := DashboardServer{
			Server: server,
			Status: status,
		}

		if statusErr != nil {
			item.Error = statusErr.Error()
		}

		items = append(items, item)
	}

	if err := tmpl.Execute(w, items); err != nil {
		log.Printf("dashboard template error: %v", err)
	}
}

func help() {
	fmt.Print(`Dinex Panel

Usage:
  dinex [command]

Commands:
  serve              Start the Dinex Panel
  admin:create       Create an administrator
  admin:list         List administrators
  version            Show version
  help               Show this help
`)
}
