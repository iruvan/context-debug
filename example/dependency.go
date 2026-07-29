package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	contextdebug "github.com/iruvan/context-debug"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Dep struct {
	DB   *sql.DB
	HTTP *http.Client
}

type UserInput struct {
	Limit int
}

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// DepDB queries all rows from the users table.
func (d *Dep) DepDB(ctx context.Context, in UserInput) (users []User, err error) {
	start := time.Now()

	defer func(start time.Time) {
		contextdebug.Collect(ctx, contextdebug.Entry{
			Name:      "DepDB.GetUsers",
			Request:   in,
			Response:  users,
			Error:     err,
			DurationMs: int(time.Since(start).Milliseconds()),
		})
	}(start)

	const query = "SELECT id, name, email, created_at FROM users LIMIT %d"

	rows, err := d.DB.QueryContext(ctx, fmt.Sprintf(query, in.Limit))
	if err != nil {
		return nil, fmt.Errorf("[Dep][DepDB] failed query to users table, %w", err)
	}

	defer rows.Close()
	for rows.Next() {
		var u User
		err = rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
		if err != nil {
			log.Printf("[Dep][DepDB] failed scan, %w", err)
			break
		}

		users = append(users, u)
	}

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

type ActivityInput struct {
	AvailabilityThreshold float64
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

func (d *Dep) DepAPI(ctx context.Context, in ActivityInput) (activity *Activity, err error) {
	start := time.Now()

	defer func(start time.Time) {
		contextdebug.Collect(ctx, contextdebug.Entry{
			Name:      "DepAPI.RandomActivity",
			Request:   in,
			Response:  activity,
			Error:     err,
			DurationMs: int(time.Since(start).Milliseconds()),
		})
	}(start)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, boredAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("[Dep][DepAPI] failed query to users table, %w", err)
	}

	var resp *http.Response
	resp, err = d.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[Dep][DepAPI] failed call, %w", err)
	}

	defer resp.Body.Close()
	var a Activity
	err = json.NewDecoder(resp.Body).Decode(&a)
	if err != nil {
		return nil, fmt.Errorf("[Dep][DepAPI] failed decode, %w", err)
	}

	activity = &a
	if a.Availability < in.AvailabilityThreshold {
		return nil, nil
	}

	return activity, err
}
