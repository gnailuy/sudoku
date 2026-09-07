package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestOpenSerializesConcurrentLegacyMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`CREATE TABLE puzzles (
		puzzle TEXT PRIMARY KEY, difficulty TEXT NOT NULL, score INTEGER NOT NULL,
		max_technique TEXT NOT NULL, source TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	const workers = 6
	start := make(chan struct{})
	opened := make(chan *DB, workers)
	errors := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			database, openErr := Open(path)
			if openErr != nil {
				errors <- openErr
				return
			}
			opened <- database
		}()
	}
	close(start)
	group.Wait()
	close(opened)
	close(errors)
	for err := range errors {
		t.Errorf("Open: %v", err)
	}
	for database := range opened {
		if err := database.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}
	if t.Failed() {
		return
	}

	check, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	columns, err := tableColumns(check.conn, "puzzles")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"play_count", "last_played_at", "completion_count", "last_completed_at"} {
		if !columns[name] {
			t.Errorf("missing migrated column %q", name)
		}
	}
}

func TestConnectionPolicyAppliesToEveryDriverConnection(t *testing.T) {
	connection, err := sql.Open("sqlite", sqliteDataSource(filepath.Join(t.TempDir(), "policy.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	connection.SetMaxOpenConns(2)

	ctx := context.Background()
	first, err := connection.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := connection.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	for index, conn := range []*sql.Conn{first, second} {
		var timeout int
		if err := conn.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&timeout); err != nil {
			t.Fatal(err)
		}
		if timeout != busyTimeoutMilliseconds {
			t.Errorf("connection %d busy_timeout = %d, want %d", index, timeout, busyTimeoutMilliseconds)
		}
	}

	wrapped, err := Open(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer wrapped.Close()
	if got := wrapped.conn.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", got)
	}
}

func TestBusyTimeoutIsBoundedAndRecovers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.db")
	holder, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	contender, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer contender.Close()

	if _, err := holder.InsertPuzzle(Puzzle{Puzzle: "shared", Difficulty: "easy", Score: 1, MaxTechnique: "single"}); err != nil {
		t.Fatal(err)
	}
	if _, err := holder.conn.Exec(`UPDATE puzzles SET play_count=2, completion_count=3 WHERE puzzle='shared'`); err != nil {
		t.Fatal(err)
	}
	locked, err := holder.conn.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Close()
	if _, err := locked.ExecContext(context.Background(), `BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	if _, err := locked.ExecContext(context.Background(), `UPDATE puzzles SET play_count=9 WHERE puzzle='shared'`); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	err = contender.ResetHistory("all", "easy")
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("expected lock exhaustion")
	}
	if elapsed < 4*time.Second || elapsed > 8*time.Second {
		t.Fatalf("lock exhaustion returned after %s, want bounded wait near five seconds", elapsed)
	}
	if _, rollbackErr := locked.ExecContext(context.Background(), `ROLLBACK`); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	var plays, completions int
	if err := contender.conn.QueryRow(`SELECT play_count, completion_count FROM puzzles WHERE puzzle='shared'`).Scan(&plays, &completions); err != nil {
		t.Fatal(err)
	}
	if plays != 2 || completions != 3 {
		t.Fatalf("failed reset changed counters: %d, %d", plays, completions)
	}
	if err := contender.ResetHistory("all", "easy"); err != nil {
		t.Fatalf("reset after lock release: %v", err)
	}
}

func TestMixedHandleWorkloadPreservesCountersSnapshotsAndIntegrity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mixed.db")
	seed, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 6 {
		puzzle := fmt.Sprintf("puzzle-%d", i)
		if _, err := seed.InsertPuzzle(Puzzle{Puzzle: puzzle, Difficulty: "easy", Score: i, MaxTechnique: "single"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}

	const handles = 4
	databases := make([]*DB, handles)
	for i := range databases {
		databases[i], err = Open(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	errors := make(chan error, handles*8)
	var group sync.WaitGroup
	for worker := range 24 {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			<-start
			database := databases[worker%handles]
			if worker%3 == 0 {
				rows, statsErr := database.PlayStatistics("")
				if statsErr != nil {
					errors <- statsErr
					return
				}
				overall := rows[len(rows)-1]
				if overall.Stored != 6 || overall.NeverSelected+overall.Selected != overall.Stored {
					errors <- fmt.Errorf("inconsistent snapshot: %+v", overall)
				}
				return
			}
			if worker%3 == 1 {
				if puzzle, acquireErr := database.AcquireForPlay("easy"); acquireErr != nil || puzzle == nil {
					errors <- fmt.Errorf("acquire: %v", acquireErr)
				}
				return
			}
			if ok, completionErr := database.RecordCompletion(fmt.Sprintf("puzzle-%d", worker%6)); completionErr != nil || !ok {
				errors <- fmt.Errorf("completion: %v", completionErr)
			}
		}(worker)
	}
	close(start)
	group.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	for _, database := range databases {
		if err := database.Close(); err != nil {
			t.Error(err)
		}
	}
	if t.Failed() {
		return
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var rows, acquisitions, completions int
	if err := reopened.conn.QueryRow(`SELECT COUNT(*), SUM(play_count), SUM(completion_count) FROM puzzles`).Scan(&rows, &acquisitions, &completions); err != nil {
		t.Fatal(err)
	}
	if rows != 6 || acquisitions != 8 || completions != 8 {
		t.Fatalf("durable totals = rows %d, acquisitions %d, completions %d", rows, acquisitions, completions)
	}
	var integrity string
	if err := reopened.conn.QueryRow(`PRAGMA quick_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if integrity != "ok" {
		t.Fatalf("quick_check = %q", integrity)
	}
}
