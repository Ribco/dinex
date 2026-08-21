package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Ribco/dinex/panel/internal/database"
)

type Auth struct {
	DB *database.DB
}

func New(db *database.DB) *Auth {
	return &Auth{DB: db}
}

func (a *Auth) CreateUser(username, password string) error {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	_, err = a.DB.SQL.Exec(`
		INSERT INTO users (username, password_hash, created_at)
		VALUES (?, ?, ?)
	`, username, string(hash), database.Now())

	return err
}

func (a *Auth) Login(username, password string) (string, error) {
	var userID int64
	var hash string

	err := a.DB.SQL.QueryRow(`
		SELECT id, password_hash
		FROM users
		WHERE username = ?
	`, username).Scan(&userID, &hash)

	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	); err != nil {
		return "", err
	}

	tokenBytes := make([]byte, 32)

	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	token := hex.EncodeToString(tokenBytes)

	_, err = a.DB.SQL.Exec(`
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`, token, userID, time.Now().Add(24*time.Hour).Unix())

	if err != nil {
		return "", err
	}

	return token, nil
}

func (a *Auth) UserFromRequest(
	r *http.Request,
) (int64, bool) {
	cookie, err := r.Cookie("dinex_session")

	if err != nil {
		return 0, false
	}

	var userID int64
	var expires int64

	err = a.DB.SQL.QueryRow(`
		SELECT user_id, expires_at
		FROM sessions
		WHERE id = ?
	`, cookie.Value).Scan(&userID, &expires)

	if err != nil {
		return 0, false
	}

	if expires < time.Now().Unix() {
		_, _ = a.DB.SQL.Exec(
			"DELETE FROM sessions WHERE id = ?",
			cookie.Value,
		)

		return 0, false
	}

	return userID, true
}

func (a *Auth) Require(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if _, ok := a.UserFromRequest(r); !ok {
			http.Redirect(
				w,
				r,
				"/login",
				http.StatusSeeOther,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
