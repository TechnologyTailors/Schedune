package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS executions (
		execution_id TEXT PRIMARY KEY,
		workload_id TEXT NOT NULL,
		node_id TEXT NOT NULL,
		state TEXT NOT NULL,
		created_at_sec INTEGER NOT NULL,
		updated_at_sec INTEGER NOT NULL,
		payload_json TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS events (
		event_id TEXT PRIMARY KEY,
		execution_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		timestamp_sec INTEGER NOT NULL,
		reason_code TEXT,
		payload_json TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_executions_node_state ON executions(node_id, state);
	CREATE INDEX IF NOT EXISTS idx_executions_workload ON executions(workload_id);
	CREATE INDEX IF NOT EXISTS idx_events_execution ON events(execution_id);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStore) SaveExecution(ctx context.Context, rec launch.LaunchExecutionRecord) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO executions (execution_id, workload_id, node_id, state, created_at_sec, updated_at_sec, payload_json)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(execution_id) DO UPDATE SET
		state = excluded.state,
		updated_at_sec = excluded.updated_at_sec,
		payload_json = excluded.payload_json
	`
	_, err = s.db.ExecContext(ctx, query,
		rec.ExecutionID,
		rec.WorkloadID,
		rec.NodeID,
		rec.State,
		rec.CreatedAtSec,
		rec.UpdatedAtSec,
		string(data),
	)
	return err
}

func (s *SQLiteStore) GetExecution(ctx context.Context, executionID string) (launch.LaunchExecutionRecord, bool, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, "SELECT payload_json FROM executions WHERE execution_id = ?", executionID).Scan(&payload)
	if err == sql.ErrNoRows {
		return launch.LaunchExecutionRecord{}, false, nil
	} else if err != nil {
		return launch.LaunchExecutionRecord{}, false, err
	}

	var rec launch.LaunchExecutionRecord
	if err := json.Unmarshal([]byte(payload), &rec); err != nil {
		return launch.LaunchExecutionRecord{}, false, err
	}

	return rec, true, nil
}

func (s *SQLiteStore) ListActiveExecutions(ctx context.Context) ([]launch.LaunchExecutionRecord, error) {
	return s.listByStateCondition(ctx, "state NOT IN ('Failed', 'Terminated', 'Exited')")
}

func (s *SQLiteStore) ListRecoverableExecutions(ctx context.Context) ([]launch.LaunchExecutionRecord, error) {
	return s.listByStateCondition(ctx, "state NOT IN ('Failed', 'Terminated', 'Exited')")
}

func (s *SQLiteStore) listByStateCondition(ctx context.Context, condition string) ([]launch.LaunchExecutionRecord, error) {
	query := fmt.Sprintf("SELECT payload_json FROM executions WHERE %s", condition)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []launch.LaunchExecutionRecord
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var rec launch.LaunchExecutionRecord
		if err := json.Unmarshal([]byte(payload), &rec); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (s *SQLiteStore) AppendEvent(ctx context.Context, ev launch.RuntimeEvent) error {
	var payload string
	if ev.PayloadJSON != nil {
		b, err := json.Marshal(ev.PayloadJSON)
		if err != nil {
			return err
		}
		payload = string(b)
	}

	query := `
	INSERT INTO events (event_id, execution_id, event_type, timestamp_sec, reason_code, payload_json)
	VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		ev.EventID,
		ev.ExecutionID,
		ev.EventType,
		ev.TimestampSec,
		ev.ReasonCode,
		payload,
	)
	return err
}

func (s *SQLiteStore) ListEvents(ctx context.Context, executionID string) ([]launch.RuntimeEvent, error) {
	query := "SELECT event_id, execution_id, event_type, timestamp_sec, reason_code, payload_json FROM events WHERE execution_id = ? ORDER BY timestamp_sec ASC"
	rows, err := s.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []launch.RuntimeEvent
	for rows.Next() {
		var ev launch.RuntimeEvent
		var payload sql.NullString
		var reasonCode sql.NullString

		if err := rows.Scan(&ev.EventID, &ev.ExecutionID, &ev.EventType, &ev.TimestampSec, &reasonCode, &payload); err != nil {
			return nil, err
		}

		if reasonCode.Valid {
			ev.ReasonCode = reasonCode.String
		}

		if payload.Valid && payload.String != "" {
			var p interface{}
			if err := json.Unmarshal([]byte(payload.String), &p); err == nil {
				ev.PayloadJSON = p
			}
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}
