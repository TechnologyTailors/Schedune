package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
)

func (s *SQLiteStore) UpsertOrphan(ctx context.Context, rec domain.OrphanRecord) error {
	argsJSON, _ := json.Marshal(rec.CommandArgs)
	detailsJSON, _ := json.Marshal(rec.Details)

	query := `
	INSERT INTO orphans (
		orphan_id, backend, pid, ppid, command, command_args_json, command_fingerprint,
		execution_id_hint, workload_id_hint, classification, status, reason_code,
		first_seen_at_sec, last_seen_at_sec, node_id, recovery_epoch, details_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(orphan_id) DO UPDATE SET
		pid = excluded.pid,
		ppid = excluded.ppid,
		classification = excluded.classification,
		status = excluded.status,
		reason_code = excluded.reason_code,
		last_seen_at_sec = excluded.last_seen_at_sec,
		recovery_epoch = excluded.recovery_epoch,
		details_json = excluded.details_json
	`
	_, err := s.db.ExecContext(ctx, query,
		rec.OrphanID, rec.Backend, rec.PID, rec.PPID, rec.Command, string(argsJSON), rec.CommandFingerprint,
		rec.ExecutionIDHint, rec.WorkloadIDHint, rec.Classification, rec.Status, rec.ReasonCode,
		rec.FirstSeenAtSec, rec.LastSeenAtSec, rec.NodeID, rec.RecoveryEpoch, string(detailsJSON),
	)
	return err
}

func (s *SQLiteStore) GetOrphan(ctx context.Context, orphanID string) (domain.OrphanRecord, bool, error) {
	query := `SELECT backend, pid, ppid, command, command_args_json, command_fingerprint,
	execution_id_hint, workload_id_hint, classification, status, reason_code,
	first_seen_at_sec, last_seen_at_sec, node_id, recovery_epoch, details_json
	FROM orphans WHERE orphan_id = ?`

	var rec domain.OrphanRecord
	rec.OrphanID = orphanID
	var argsJSON, detailsJSON sql.NullString
	var ppid sql.NullInt64
	var execHint, workHint, nodeID, epoch sql.NullString

	err := s.db.QueryRowContext(ctx, query, orphanID).Scan(
		&rec.Backend, &rec.PID, &ppid, &rec.Command, &argsJSON, &rec.CommandFingerprint,
		&execHint, &workHint, &rec.Classification, &rec.Status, &rec.ReasonCode,
		&rec.FirstSeenAtSec, &rec.LastSeenAtSec, &nodeID, &epoch, &detailsJSON,
	)

	if err == sql.ErrNoRows {
		return rec, false, nil
	} else if err != nil {
		return rec, false, err
	}

	if ppid.Valid {
		pidVal := int(ppid.Int64)
		rec.PPID = &pidVal
	}
	if argsJSON.Valid {
		_ = json.Unmarshal([]byte(argsJSON.String), &rec.CommandArgs)
	}
	if detailsJSON.Valid {
		_ = json.Unmarshal([]byte(detailsJSON.String), &rec.Details)
	}
	rec.ExecutionIDHint = execHint.String
	rec.WorkloadIDHint = workHint.String
	rec.NodeID = nodeID.String
	rec.RecoveryEpoch = epoch.String

	return rec, true, nil
}

func (s *SQLiteStore) ListOrphans(ctx context.Context, filter domain.OrphanFilter) ([]domain.OrphanRecord, error) {
	query := `SELECT orphan_id, backend, pid, ppid, command, command_args_json, command_fingerprint,
	execution_id_hint, workload_id_hint, classification, status, reason_code,
	first_seen_at_sec, last_seen_at_sec, node_id, recovery_epoch, details_json
	FROM orphans`
	
	var conditions []string
	var args []interface{}

	if filter.Backend != "" {
		conditions = append(conditions, "backend = ?")
		args = append(args, filter.Backend)
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Classification != "" {
		conditions = append(conditions, "classification = ?")
		args = append(args, filter.Classification)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.OrphanRecord
	for rows.Next() {
		var rec domain.OrphanRecord
		var argsJSON, detailsJSON sql.NullString
		var ppid sql.NullInt64
		var execHint, workHint, nodeID, epoch sql.NullString

		if err := rows.Scan(&rec.OrphanID, &rec.Backend, &rec.PID, &ppid, &rec.Command, &argsJSON, &rec.CommandFingerprint,
			&execHint, &workHint, &rec.Classification, &rec.Status, &rec.ReasonCode,
			&rec.FirstSeenAtSec, &rec.LastSeenAtSec, &nodeID, &epoch, &detailsJSON); err != nil {
			return nil, err
		}
		if ppid.Valid {
			pidVal := int(ppid.Int64)
			rec.PPID = &pidVal
		}
		if argsJSON.Valid {
			_ = json.Unmarshal([]byte(argsJSON.String), &rec.CommandArgs)
		}
		if detailsJSON.Valid {
			_ = json.Unmarshal([]byte(detailsJSON.String), &rec.Details)
		}
		rec.ExecutionIDHint = execHint.String
		rec.WorkloadIDHint = workHint.String
		rec.NodeID = nodeID.String
		rec.RecoveryEpoch = epoch.String

		records = append(records, rec)
	}

	return records, rows.Err()
}

func (s *SQLiteStore) MarkMissingAsDisappeared(ctx context.Context, observedOrphanIDs []string, nowSec int64) error {
	// If it was Active but not observed in this sweep, mark as Disappeared.
	query := "UPDATE orphans SET status = 'Disappeared', last_seen_at_sec = ? WHERE status = 'Active'"
	var args []interface{}
	args = append(args, nowSec)

	if len(observedOrphanIDs) > 0 {
		placeholders := make([]string, len(observedOrphanIDs))
		for i, id := range observedOrphanIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query += fmt.Sprintf(" AND orphan_id NOT IN (%s)", strings.Join(placeholders, ","))
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}
