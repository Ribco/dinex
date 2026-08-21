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

	"github.com/Ribco/dinex/panel/internal/auth"
	"github.com/Ribco/dinex/panel/internal/database"
)

const version = "0.1.0"

type App struct {
	DB   *database.DB
	Auth *auth.Auth
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
		DB:   db,
		Auth: auth.New(db),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	mux.HandleFunc("/health", health)
	mux.HandleFunc("/login", app.login)

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
		http.Error(w, err.Error(), 500)
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
		http.Error(w, err.Error(), 500)
		return
	}

	_ = tmpl.Execute(w, nil)
}

func help() {
	fmt.Println(`Dinex Panel

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
