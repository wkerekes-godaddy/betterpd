package blame

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"betterpd/internal/config"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

type StoredEntry struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

type Change struct {
	ID             int64
	ScheduleID     string
	ScheduleName   string
	DetectedAt     time.Time
	ChangeType     string // added, removed, modified, override
	Description    string
	OriginalUser   string
	OriginalUserID string
	NewUser        string
	NewUserID      string
	Start          time.Time
	End            time.Time
	ChangedBy      string
}

func OpenDB() (*DB, error) {
	dbPath := filepath.Join(config.ConfigDir(), "blame.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening blame database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS snapshots (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			schedule_id   TEXT NOT NULL,
			schedule_name TEXT NOT NULL,
			fetched_at    TEXT NOT NULL,
			entries_json  TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_snapshots_schedule ON snapshots(schedule_id, fetched_at DESC);

		CREATE TABLE IF NOT EXISTS changes (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			schedule_id      TEXT NOT NULL,
			schedule_name    TEXT NOT NULL,
			detected_at      TEXT NOT NULL,
			change_type      TEXT NOT NULL,
			description      TEXT NOT NULL,
			original_user    TEXT NOT NULL DEFAULT '',
			original_user_id TEXT NOT NULL DEFAULT '',
			new_user         TEXT NOT NULL DEFAULT '',
			new_user_id      TEXT NOT NULL DEFAULT '',
			start_time       TEXT NOT NULL DEFAULT '',
			end_time         TEXT NOT NULL DEFAULT '',
			changed_by       TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_changes_detected ON changes(detected_at DESC);

		CREATE TABLE IF NOT EXISTS sync_meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("migrating blame database: %w", err)
	}

	// Add user ID columns to existing databases
	for _, col := range []string{"original_user_id", "new_user_id"} {
		db.Exec(fmt.Sprintf(`ALTER TABLE changes ADD COLUMN %s TEXT NOT NULL DEFAULT ''`, col))
	}

	return nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) SaveSnapshot(scheduleID, scheduleName string, entries []StoredEntry) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(
		`INSERT INTO snapshots (schedule_id, schedule_name, fetched_at, entries_json) VALUES (?, ?, ?, ?)`,
		scheduleID, scheduleName, time.Now().UTC().Format(time.RFC3339), string(data),
	)
	return err
}

func (d *DB) GetLastSnapshot(scheduleID string) ([]StoredEntry, error) {
	var raw string
	err := d.db.QueryRow(
		`SELECT entries_json FROM snapshots WHERE schedule_id = ? ORDER BY fetched_at DESC LIMIT 1`,
		scheduleID,
	).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []StoredEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (d *DB) RecordChange(c Change) error {
	_, err := d.db.Exec(
		`INSERT INTO changes (schedule_id, schedule_name, detected_at, change_type, description, original_user, original_user_id, new_user, new_user_id, start_time, end_time, changed_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ScheduleID, c.ScheduleName,
		c.DetectedAt.UTC().Format(time.RFC3339),
		c.ChangeType, c.Description,
		c.OriginalUser, c.OriginalUserID,
		c.NewUser, c.NewUserID,
		formatTime(c.Start), formatTime(c.End),
		c.ChangedBy,
	)
	return err
}

func (d *DB) ListChanges(since time.Time) ([]Change, error) {
	rows, err := d.db.Query(
		`SELECT id, schedule_id, schedule_name, detected_at, change_type, description, original_user, original_user_id, new_user, new_user_id, start_time, end_time, changed_by
		 FROM changes WHERE detected_at >= ? ORDER BY detected_at ASC`,
		since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []Change
	for rows.Next() {
		var c Change
		var detectedAt, startTime, endTime string
		if err := rows.Scan(&c.ID, &c.ScheduleID, &c.ScheduleName, &detectedAt, &c.ChangeType, &c.Description, &c.OriginalUser, &c.OriginalUserID, &c.NewUser, &c.NewUserID, &startTime, &endTime, &c.ChangedBy); err != nil {
			return nil, err
		}
		c.DetectedAt, _ = time.Parse(time.RFC3339, detectedAt)
		c.Start, _ = time.Parse(time.RFC3339, startTime)
		c.End, _ = time.Parse(time.RFC3339, endTime)
		changes = append(changes, c)
	}
	return changes, rows.Err()
}

func (d *DB) GetLastSyncTime() (time.Time, error) {
	var val string
	err := d.db.QueryRow(`SELECT value FROM sync_meta WHERE key = 'last_sync_at'`).Scan(&val)
	if err == sql.ErrNoRows {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, val)
}

func (d *DB) SetLastSyncTime(t time.Time) error {
	_, err := d.db.Exec(
		`INSERT INTO sync_meta (key, value) VALUES ('last_sync_at', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		t.UTC().Format(time.RFC3339),
	)
	return err
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
