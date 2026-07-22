package contextdebug_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	contextdebug "github.com/iruvan/context-debug"
)

type unrelatedKey struct{}

// exportedFieldsErr is an error implementation whose message survives JSON
// marshaling only because its fields happen to be exported, unlike errors
// from errors.New/fmt.Errorf.
type exportedFieldsErr struct {
	Msg string
}

func (e *exportedFieldsErr) Error() string { return e.Msg }

func TestNew(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
	}{
		{"background", context.Background()},
		{"todo", context.TODO()},
		{"already carrying an unrelated value", context.WithValue(context.Background(), unrelatedKey{}, "v")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := contextdebug.New(tc.ctx)

			if !contextdebug.Enabled(ctx) {
				t.Fatal("Enabled(New(ctx)) = false, want true")
			}
			if got := contextdebug.Snapshot(ctx); got == nil || len(got) != 0 {
				t.Fatalf("Snapshot(New(ctx)) = %#v, want an empty non-nil slice", got)
			}
		})
	}

	t.Run("nil context panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("New(nil) did not panic, want panic")
			}
		}()
		var nilCtx context.Context
		_ = contextdebug.New(nilCtx)
	})

	t.Run("nested New shadows the outer store (documents current behavior)", func(t *testing.T) {
		outer := contextdebug.New(context.Background())
		contextdebug.Collect(outer, contextdebug.Entry{Name: "outer"})

		inner := contextdebug.New(outer)
		contextdebug.Collect(inner, contextdebug.Entry{Name: "inner"})

		got := contextdebug.Snapshot(inner)
		want := []contextdebug.Entry{{Name: "inner"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Snapshot(inner) = %#v, want %#v (calling New twice replaces the store rather than merging)", got, want)
		}
	})
}

func TestEnabled(t *testing.T) {
	cases := []struct {
		name string
		ctx  func() context.Context
		want bool
	}{
		{"background", func() context.Context { return context.Background() }, false},
		{"todo", func() context.Context { return context.TODO() }, false},
		{"after New", func() context.Context { return contextdebug.New(context.Background()) }, true},
		{
			name: "context derived from a New'd context via WithCancel stays enabled",
			ctx: func() context.Context {
				ctx := contextdebug.New(context.Background())
				derived, cancel := context.WithCancel(ctx)
				cancel()
				return derived
			},
			want: true,
		},
		{
			name: "context derived from a New'd context via WithValue stays enabled",
			ctx: func() context.Context {
				ctx := contextdebug.New(context.Background())
				return context.WithValue(ctx, unrelatedKey{}, "v")
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := contextdebug.Enabled(tc.ctx()); got != tc.want {
				t.Fatalf("Enabled() = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("nil context panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Enabled(nil) did not panic, want panic")
			}
		}()
		var nilCtx context.Context
		_ = contextdebug.Enabled(nilCtx)
	})
}

func TestCollect(t *testing.T) {
	errBoom := errors.New("boom")

	cases := []struct {
		name    string
		ctx     func() context.Context
		entries []contextdebug.Entry
		enabled bool
	}{
		{
			name:    "no store: collect is a silent no-op",
			ctx:     func() context.Context { return context.Background() },
			entries: []contextdebug.Entry{{Name: "a"}},
			enabled: false,
		},
		{
			name: "single entry round-trips",
			ctx:  func() context.Context { return contextdebug.New(context.Background()) },
			entries: []contextdebug.Entry{
				{Name: "DepDB", Request: "SELECT 1", Response: 42, LatencyMs: 5},
			},
			enabled: true,
		},
		{
			name: "multiple entries preserve insertion order",
			ctx:  func() context.Context { return contextdebug.New(context.Background()) },
			entries: []contextdebug.Entry{
				{Name: "first", LatencyMs: 1},
				{Name: "second", LatencyMs: 2},
				{Name: "third", LatencyMs: 3},
			},
			enabled: true,
		},
		{
			name: "entry with a non-nil error round-trips",
			ctx:  func() context.Context { return contextdebug.New(context.Background()) },
			entries: []contextdebug.Entry{
				{Name: "failed-call", Error: errBoom},
			},
			enabled: true,
		},
		{
			name: "entry with zero-value request/response/error round-trips",
			ctx:  func() context.Context { return contextdebug.New(context.Background()) },
			entries: []contextdebug.Entry{
				{Name: "empty"},
			},
			enabled: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx()
			for _, e := range tc.entries {
				contextdebug.Collect(ctx, e)
			}

			got := contextdebug.Snapshot(ctx)
			if !tc.enabled {
				if len(got) != 0 {
					t.Fatalf("Snapshot() = %#v, want empty (debug was never enabled)", got)
				}
				return
			}

			if !reflect.DeepEqual(got, tc.entries) {
				t.Fatalf("Snapshot() = %#v, want %#v", got, tc.entries)
			}
		})
	}

	t.Run("nil context panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Collect(nil, ...) did not panic, want panic")
			}
		}()
		var nilCtx context.Context
		contextdebug.Collect(nilCtx, contextdebug.Entry{Name: "x"})
	})

	t.Run("concurrent collect is race-safe and loses no entries", func(t *testing.T) {
		ctx := contextdebug.New(context.Background())
		const n = 200

		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(i int) {
				defer wg.Done()
				contextdebug.Collect(ctx, contextdebug.Entry{Name: "c", LatencyMs: i})
			}(i)
		}
		wg.Wait()

		if got := contextdebug.Snapshot(ctx); len(got) != n {
			t.Fatalf("Snapshot() len = %d, want %d", len(got), n)
		}
	})

	t.Run("interface{} fields are shallow-copied (documents current behavior)", func(t *testing.T) {
		ctx := contextdebug.New(context.Background())
		req := &struct{ N int }{N: 1}

		contextdebug.Collect(ctx, contextdebug.Entry{Name: "shallow", Request: req})
		req.N = 2 // mutate after Collect

		got := contextdebug.Snapshot(ctx)
		if len(got) != 1 {
			t.Fatalf("Snapshot() len = %d, want 1", len(got))
		}
		stored, ok := got[0].Request.(*struct{ N int })
		if !ok {
			t.Fatalf("Snapshot()[0].Request = %#v, want *struct{ N int }", got[0].Request)
		}
		if stored.N != 2 {
			t.Fatalf("stored.N = %d, want 2 (Collect stores the pointer, not a deep copy, so post-Collect mutation is visible in the snapshot)", stored.N)
		}
	})
}

func TestSnapshot(t *testing.T) {
	t.Run("no store returns nil", func(t *testing.T) {
		if got := contextdebug.Snapshot(context.Background()); got != nil {
			t.Fatalf("Snapshot() = %#v, want nil", got)
		}
	})

	t.Run("store with zero entries returns an empty non-nil slice", func(t *testing.T) {
		ctx := contextdebug.New(context.Background())
		got := contextdebug.Snapshot(ctx)
		if got == nil || len(got) != 0 {
			t.Fatalf("Snapshot() = %#v, want an empty non-nil slice", got)
		}
	})

	t.Run("returned slice is a copy: local mutation does not affect the store", func(t *testing.T) {
		ctx := contextdebug.New(context.Background())
		contextdebug.Collect(ctx, contextdebug.Entry{Name: "original"})

		snap1 := contextdebug.Snapshot(ctx)
		snap1 = append(snap1, contextdebug.Entry{Name: "appended-locally"})
		snap1[0].Name = "tampered"

		snap2 := contextdebug.Snapshot(ctx)
		want := []contextdebug.Entry{{Name: "original"}}
		if !reflect.DeepEqual(snap2, want) {
			t.Fatalf("Snapshot() after local mutation = %#v, want untouched %#v", snap2, want)
		}
	})

	t.Run("nil context panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Snapshot(nil) did not panic, want panic")
			}
		}()
		var nilCtx context.Context
		_ = contextdebug.Snapshot(nilCtx)
	})
}

func TestEntry_ErrorJSONMarshaling(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil error marshals to null", nil, `{"name":"e","req":null,"resp":null,"err":null,"lat_ms":0}`},
		{
			name: "errors.New: message is lost (documents current behavior)",
			err:  errors.New("boom"),
			want: `{"name":"e","req":null,"resp":null,"err":{},"lat_ms":0}`,
		},
		{
			name: "fmt.Errorf wrapped error: message is lost (documents current behavior)",
			err:  fmt.Errorf("wrap: %w", errors.New("boom")),
			want: `{"name":"e","req":null,"resp":null,"err":{},"lat_ms":0}`,
		},
		{
			name: "error with exported fields: those fields happen to survive",
			err:  &exportedFieldsErr{Msg: "boom"},
			want: `{"name":"e","req":null,"resp":null,"err":{"Msg":"boom"},"lat_ms":0}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(contextdebug.Entry{Name: "e", Error: tc.err})
			if err != nil {
				t.Fatalf("json.Marshal() error = %v, want nil", err)
			}
			if got := string(b); got != tc.want {
				t.Fatalf("json.Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}
