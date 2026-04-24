package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain/lifecycle"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type DurableExecutionStore interface {
	SaveExecution(ctx context.Context, rec launch.LaunchExecutionRecord) error
	GetExecution(ctx context.Context, executionID string) (launch.LaunchExecutionRecord, bool, error)
	ListActiveExecutions(ctx context.Context) ([]launch.LaunchExecutionRecord, error)
	ListRecoverableExecutions(ctx context.Context) ([]launch.LaunchExecutionRecord, error)
}

type DurableEventStore interface {
	AppendEvent(ctx context.Context, ev launch.RuntimeEvent) error
	ListEvents(ctx context.Context, executionID string) ([]launch.RuntimeEvent, error)
}

type RecoveryBootstrapper struct {
	ExecStore   DurableExecutionStore
	EventStore  DurableEventStore
	Inspector   inspect.Inspector
	Reassociate *ReassociationEngine
	Epoch       string
}

func NewRecoveryBootstrapper(execStore DurableExecutionStore, eventStore DurableEventStore, inspector inspect.Inspector) *RecoveryBootstrapper {
	return &RecoveryBootstrapper{
		ExecStore:   execStore,
		EventStore:  eventStore,
		Inspector:   inspector,
		Reassociate: NewReassociationEngine(),
		Epoch:       uuid.New().String(),
	}
}

func (b *RecoveryBootstrapper) Bootstrap(ctx context.Context) error {
	log.Info().Str("recovery_epoch", b.Epoch).Msg("Starting control-plane recovery bootstrap")

	recs, err := b.ExecStore.ListRecoverableExecutions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list recoverable executions: %v", err)
	}

	for _, rec := range recs {
		b.recoverExecution(ctx, rec)
	}

	log.Info().Int("recovered_count", len(recs)).Msg("Bootstrap recovery complete")
	return nil
}

func (b *RecoveryBootstrapper) recoverExecution(ctx context.Context, rec launch.LaunchExecutionRecord) {
	log.Info().Str("execution_id", rec.ExecutionID).Str("state", string(rec.State)).Msg("Recovering execution")
	rec.RecoveryEpoch = b.Epoch
	now := time.Now().Unix()
	rec.LastObservedAtSec = &now

	var eventType string
	var reasonCode string
	var msg string

	if rec.PID != nil {
		obs, err := b.Inspector.Inspect(*rec.PID)
		if err != nil {
			lifecycle.TransitionTo(&rec, launch.StateUnknown, "ERR_RECOVERY_REHYDRATE_FAILED", fmt.Sprintf("Failed to inspect PID: %v", err))
			eventType = "ExecutionRecoveryUnknown"
			reasonCode = "ERR_RECOVERY_REHYDRATE_FAILED"
			msg = "Process inspect failed"
		} else if obs.ProcessExists {
			// For V1, we assume the PID is enough if it exists. In V2, we'd use process_identity
			// and ReassociationEngine.
			if rec.State == launch.StateTerminating {
				msg = "Process still exists, remaining in Terminating"
			} else {
				msg = "Process confirmed alive"
			}
			eventType = "ExecutionRehydrated"
			reasonCode = "RECOVERY_CONFIRMED"
			// Force liveness update
			rec.RuntimeLiveness = "Alive"
			lifecycle.AppendTrace(&rec, "RecoveryBootstrap", "Success", reasonCode, msg)
		} else {
			// Process missing
			if rec.State == launch.StateTerminating {
				lifecycle.TransitionTo(&rec, launch.StateTerminated, "ERR_RECOVERY_EXECUTION_MISSING", "Terminating process missing on restart")
				eventType = "ExecutionRehydrated"
				reasonCode = "RECOVERY_TERMINATED_MISSING"
				msg = "Terminating process converged to Terminated"
			} else {
				lifecycle.TransitionTo(&rec, launch.StateUnknown, "ERR_RECOVERY_EXECUTION_MISSING", "Process missing on restart")
				eventType = "ExecutionRecoveryUnknown"
				reasonCode = "ERR_RECOVERY_EXECUTION_MISSING"
				msg = "Process missing"
			}
		}
	} else {
		// Mid-flight before PID
		lifecycle.TransitionTo(&rec, launch.StateUnknown, "ERR_RECOVERY_STALE_HANDLE", "Mid-flight state without PID recovered as Unknown")
		eventType = "ExecutionRecoveryUnknown"
		reasonCode = "ERR_RECOVERY_STALE_HANDLE"
		msg = "No PID assigned before crash"
	}

	b.ExecStore.SaveExecution(ctx, rec)
	b.EventStore.AppendEvent(ctx, launch.RuntimeEvent{
		EventID:      uuid.New().String(),
		ExecutionID:  rec.ExecutionID,
		EventType:    eventType,
		TimestampSec: now,
		ReasonCode:   reasonCode,
		PayloadJSON:  map[string]string{"message": msg},
	})
}
