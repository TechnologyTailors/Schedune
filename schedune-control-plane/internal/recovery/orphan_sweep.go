package recovery

import (
	"context"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type DurableOrphanStore interface {
	UpsertOrphan(ctx context.Context, rec domain.OrphanRecord) error
	GetOrphan(ctx context.Context, orphanID string) (domain.OrphanRecord, bool, error)
	ListOrphans(ctx context.Context, filter domain.OrphanFilter) ([]domain.OrphanRecord, error)
	MarkMissingAsDisappeared(ctx context.Context, observedOrphanIDs []string, nowSec int64) error
}

type OrphanSweepService struct {
	Enumerator  runtime.Enumerator
	ExecStore   DurableExecutionStore
	OrphanStore DurableOrphanStore
	EventStore  DurableEventStore
	NodeID      string
}

func NewOrphanSweepService(enumerator runtime.Enumerator, execStore DurableExecutionStore, orphanStore DurableOrphanStore, eventStore DurableEventStore, nodeID string) *OrphanSweepService {
	return &OrphanSweepService{
		Enumerator:  enumerator,
		ExecStore:   execStore,
		OrphanStore: orphanStore,
		EventStore:  eventStore,
		NodeID:      nodeID,
	}
}

func (s *OrphanSweepService) SweepOnce(ctx context.Context) error {
	log.Debug().Msg("Starting orphan sweep")
	nowSec := time.Now().Unix()

	// 1. Enumerate runtime processes
	processes, err := s.Enumerator.Enumerate()
	if err != nil {
		return err
	}

	// 2. Load recoverable executions
	execs, err := s.ExecStore.ListRecoverableExecutions(ctx)
	if err != nil {
		return err
	}

	var observedOrphanIDs []string

	// 3. Match and Classify
	for _, p := range processes {
		match := MatchEnumeratedProcessToExecution(p, execs)
		if match.Matched {
			// Not an orphan
			continue
		}

		orphan := ClassifyOrphan(p, match)
		orphan.NodeID = s.NodeID

		// Load existing to preserve FirstSeenAtSec
		existing, found, err := s.OrphanStore.GetOrphan(ctx, orphan.OrphanID)
		if err == nil && found {
			orphan.FirstSeenAtSec = existing.FirstSeenAtSec

			// Only emit event if classification or status changed significantly
			if existing.Classification != orphan.Classification || existing.Status != orphan.Status {
				s.emitEvent(ctx, "OrphanReclassified", orphan)
			}
		} else {
			// Newly detected
			s.emitEvent(ctx, "OrphanDetected", orphan)
		}

		err = s.OrphanStore.UpsertOrphan(ctx, orphan)
		if err != nil {
			log.Error().Err(err).Str("orphan_id", orphan.OrphanID).Msg("Failed to upsert orphan")
		} else {
			observedOrphanIDs = append(observedOrphanIDs, orphan.OrphanID)
		}
	}

	// 4. Mark disappearances
	// We pass the IDs we *did* observe this round. Anything Active not in this list becomes Disappeared.
	err = s.OrphanStore.MarkMissingAsDisappeared(ctx, observedOrphanIDs, nowSec)
	if err != nil {
		log.Error().Err(err).Msg("Failed to mark missing orphans as disappeared")
	}

	// In a real implementation, we would query the newly Disappeared ones and emit OrphanDisappeared events.

	log.Debug().Int("observed_orphans", len(observedOrphanIDs)).Msg("Orphan sweep completed")
	return nil
}

func (s *OrphanSweepService) emitEvent(ctx context.Context, eventType string, orphan domain.OrphanRecord) {
	ev := launch.RuntimeEvent{
		EventID:      uuid.New().String(),
		ExecutionID:  orphan.ExecutionIDHint, // Can be empty if unmanaged
		EventType:    eventType,
		TimestampSec: orphan.LastSeenAtSec,
		ReasonCode:   orphan.ReasonCode,
		PayloadJSON:  orphan,
	}
	s.EventStore.AppendEvent(ctx, ev)
}
