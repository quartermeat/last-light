package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Event struct {
	Hour    float64 `json:"hour"`
	Kind    string  `json:"kind"`
	Details string  `json:"details"`
}
type Run struct {
	Name   string  `json:"name"`
	Hours  int     `json:"hours"`
	Events []Event `json:"events"`
}
type Entry struct {
	Rank      int    `json:"rank"`
	Name      string `json:"name"`
	Hours     int    `json:"hours"`
	CreatedAt string `json:"created_at"`
}

var initials = regexp.MustCompile(`^[A-Z0-9]{3}$`)

func main() {
	dbPath := os.Getenv("LAST_LIGHT_DB")
	if dbPath == "" {
		dbPath = "leaderboard.db"
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`PRAGMA journal_mode=WAL; CREATE TABLE IF NOT EXISTS runs (id INTEGER PRIMARY KEY, name TEXT NOT NULL, hours INTEGER NOT NULL, created_at TEXT NOT NULL); CREATE TABLE IF NOT EXISTS events (run_id INTEGER NOT NULL, hour REAL NOT NULL, kind TEXT NOT NULL, details TEXT NOT NULL, FOREIGN KEY(run_id) REFERENCES runs(id)); CREATE INDEX IF NOT EXISTS runs_hours_idx ON runs(hours DESC); CREATE INDEX IF NOT EXISTS events_run_idx ON events(run_id);`); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/leaderboard", leaderboardHandler(db))
	mux.HandleFunc("POST /api/runs", runHandler(db))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	webRoot := os.Getenv("LAST_LIGHT_WEB_ROOT")
	if webRoot == "" {
		if _, statErr := os.Stat("web"); statErr == nil {
			webRoot = "web"
		} else {
			webRoot = "../web"
		}
	}
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("Last Light server listening on %s (web root: %s)", addr, webRoot)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

func leaderboardHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		rows, err := db.Query(`SELECT name, hours, created_at FROM runs ORDER BY hours DESC, id ASC LIMIT 100`)
		if err != nil {
			http.Error(w, "database error", 500)
			return
		}
		defer rows.Close()
		entries := make([]Entry, 0, 100)
		rank := 1
		for rows.Next() {
			var e Entry
			if err := rows.Scan(&e.Name, &e.Hours, &e.CreatedAt); err != nil {
				http.Error(w, "database error", 500)
				return
			}
			e.Rank = rank
			rank++
			entries = append(entries, e)
		}
		writeJSON(w, entries)
	}
}

func runHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var run Run
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if json.NewDecoder(r.Body).Decode(&run) != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		run.Name = strings.ToUpper(strings.TrimSpace(run.Name))
		if !initials.MatchString(run.Name) {
			http.Error(w, "name must be exactly 3 letters or digits", 400)
			return
		}
		if run.Hours <= 0 || run.Hours > 100000 {
			http.Error(w, "hours out of range", 400)
			return
		}
		if len(run.Events) > 2000 {
			http.Error(w, "too many events", 400)
			return
		}
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "database error", 500)
			return
		}
		defer tx.Rollback()
		result, err := tx.Exec(`INSERT INTO runs(name, hours, created_at) VALUES(?, ?, ?)`, run.Name, run.Hours, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			http.Error(w, "database error", 500)
			return
		}
		id, _ := result.LastInsertId()
		stmt, err := tx.Prepare(`INSERT INTO events(run_id, hour, kind, details) VALUES(?, ?, ?, ?)`)
		if err != nil {
			http.Error(w, "database error", 500)
			return
		}
		defer stmt.Close()
		for _, event := range run.Events {
			if event.Kind == "" || len(event.Kind) > 64 || len(event.Details) > 500 {
				http.Error(w, "invalid event", 400)
				return
			}
			if _, err = stmt.Exec(id, event.Hour, event.Kind, event.Details); err != nil {
				http.Error(w, "database error", 500)
				return
			}
		}
		if err = tx.Commit(); err != nil {
			http.Error(w, "database error", 500)
			return
		}
		writeJSON(w, map[string]any{"ok": true, "id": id})
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

var _ = errors.New
