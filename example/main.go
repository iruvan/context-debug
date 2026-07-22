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
	_ "github.com/jackc/pgx/v5/stdlib"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Activity struct {
	Activity      string  `json:"activity"`
	Availability  float64 `json:"availability"`
	Type          string  `json:"type"`
	Participants  int     `json:"participants"`
	Price         float64 `json:"price"`
	Accessibility string  `json:"accessibility"`
	Duration      string  `json:"duration"`
	KidFriendly   bool    `json:"kidFriendly"`
	Link          string  `json:"link"`
	Key           string  `json:"key"`
}

type Result struct {
	Users    []User    `json:"users"`
	Activity *Activity `json:"activity,omitempty"`
}

type response struct {
	Data  Result               `json:"data"`
	Debug []contextdebug.Entry `json:"debug,omitempty"`
}

const boredAPIURL = "https://bored-api.appbrewery.com/random"

func main() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres@localhost:5433/postgres?sslmode=disable"
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /test-1", Handler(db, client))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func Handler(db *sql.DB, client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		debugEnabled := r.Header.Get("is-debug") == "1"
		if debugEnabled {
			ctx = contextdebug.New(ctx)
		}

		result, err := Usecase(ctx, db, client)

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

func Usecase(ctx context.Context, db *sql.DB, client *http.Client) (Result, error) {
	users, err := DepDB(ctx, db)
	if err != nil {
		return Result{}, err
	}
	activity, err := DepAPI(ctx, client)
	if err != nil {
		return Result{}, err
	}
	return Result{Users: users, Activity: activity}, nil
}

// DepDB queries all rows from the users table.
func DepDB(ctx context.Context, db *sql.DB) ([]User, error) {
	start := time.Now()
	const query = "SELECT id, name, email, created_at FROM users"

	var users []User
	rows, err := db.QueryContext(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var u User
			if err = rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
				break
			}
			users = append(users, u)
		}
		if err == nil {
			err = rows.Err()
		}
	}

	contextdebug.Collect(ctx, contextdebug.Entry{
		Name:      "DepDB.GetUsers",
		Request:   query,
		Response:  users,
		Error:     err,
		LatencyMs: int(time.Since(start).Milliseconds()),
	})
	return users, err
}

// DepAPI will hit https://bored-api.appbrewery.com/random and get `activity`
// Sample response:
//
//	{
//	  "activity": "Solve a Rubik's cube",
//	  "availability": 0.1,
//	  "type": "recreational",
//	  "participants": 1,
//	  "price": 0,
//	  "accessibility": "Few to no challenges",
//	  "duration": "hours",
//	  "kidFriendly": true,
//	  "link": "",
//	  "key": "4151544"
//	}
func DepAPI(ctx context.Context, client *http.Client) (*Activity, error) {
	start := time.Now()

	var activity *Activity
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, boredAPIURL, nil)
	if err == nil {
		var resp *http.Response
		resp, err = client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			var a Activity
			if err = json.NewDecoder(resp.Body).Decode(&a); err == nil {
				activity = &a
			}
		}
	}

	contextdebug.Collect(ctx, contextdebug.Entry{
		Name:      "DepAPI.RandomActivity",
		Request:   boredAPIURL,
		Response:  activity,
		Error:     err,
		LatencyMs: int(time.Since(start).Milliseconds()),
	})
	return activity, err
}
