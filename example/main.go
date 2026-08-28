package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	contextdebug "github.com/iruvan/context-debug"
)

type response struct {
	Data  Result               `json:"data"`
	Debug []contextdebug.Entry `json:"debug,omitempty"`
}

const boredAPIURL = "https://bored-api.appbrewery.com/random"

func main() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://demo:demo@localhost:5433/contextdebug?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	dep := &Dep{
		DB:   db,
		HTTP: client,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /test-1", dep.Handler())

	log.Println("listening on :8023")
	log.Fatal(http.ListenAndServe(":8023", mux))
}

func (d *Dep) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		debugEnabled := r.Header.Get("is-debug") == "1"
		if debugEnabled {
			ctx = contextdebug.New(ctx)
		}

		result, err := d.Usecase(ctx)

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		resp := response{Data: result}
		if debugEnabled {
			resp.Debug = contextdebug.Snapshot(ctx)
		}
		json.NewEncoder(w).Encode(resp)
	}
}

type Result struct {
	Users    []User    `json:"users"`
	Activity *Activity `json:"activity,omitempty"`
}

func (d *Dep) Usecase(ctx context.Context) (Result, error) {
	users, err := d.DepDB(ctx, UserInput{
		Limit: 1,
	})
	if err != nil {
		return Result{}, err
	}

	activity, err := d.DepAPI(ctx, ActivityInput{
		AvailabilityThreshold: 0.1,
	})
	if err != nil {
		return Result{}, err
	}

	return Result{Users: users, Activity: activity}, nil
}
