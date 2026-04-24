package store

import (
	"context"
	"errors"
	"sync"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/domain"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/launch"
)

type InMemoryStore struct {
	mu           sync.RWMutex
	nodes        map[string]domain.NodeRecord
	rawEnvelopes map[string]schema.SchedulerEnvelope
	executions   map[string]launch.LaunchExecutionRecord
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		nodes:        make(map[string]domain.NodeRecord),
		rawEnvelopes: make(map[string]schema.SchedulerEnvelope),
		executions:   make(map[string]launch.LaunchExecutionRecord),
	}
}

func (s *InMemoryStore) SaveExecution(ctx context.Context, rec launch.LaunchExecutionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[rec.ExecutionID] = rec
	return nil
}

func (s *InMemoryStore) GetExecution(ctx context.Context, id string) (launch.LaunchExecutionRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.executions[id]
	if !ok {
		return launch.LaunchExecutionRecord{}, false, errors.New("execution not found")
	}
	return rec, true, nil
}

func (s *InMemoryStore) SaveNodeState(env schema.SchedulerEnvelope, record domain.NodeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rawEnvelopes[env.NodeID] = env
	s.nodes[record.ID] = record
	return nil
}

func (s *InMemoryStore) GetNode(id string) (domain.NodeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if node, exists := s.nodes[id]; exists {
		return node, nil
	}
	return domain.NodeRecord{}, errors.New("node not found")
}

func (s *InMemoryStore) ListNodesByCompatibility(class string) []domain.NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []domain.NodeRecord
	for _, node := range s.nodes {
		if node.Compatibility.Class == class {
			matches = append(matches, node)
		}
	}
	return matches
}

func (s *InMemoryStore) ListNodesEligibleFor(runtimeClass string) []domain.NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []domain.NodeRecord
	for _, node := range s.nodes {
		engine := domain.NewEligibilityEngine(node)

		switch runtimeClass {
		case "kvm":
			if engine.CanRunKVMVMs() {
				matches = append(matches, node)
			}
		case "firecracker":
			if engine.CanRunFirecrackerMicroVMs() {
				matches = append(matches, node)
			}
		case "arm-prod":
			if engine.CanJoinArmProductionPool() {
				matches = append(matches, node)
			}
		case "x86-holding":
			if engine.IsOnlyEligibleForHoldingPool() {
				matches = append(matches, node)
			}
		}
	}
	return matches
}

func (s *InMemoryStore) ListStaleNodes() []domain.NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []domain.NodeRecord
	for _, node := range s.nodes {
		if node.Freshness.IsStale {
			matches = append(matches, node)
		}
	}
	return matches
}

func (s *InMemoryStore) ListAllNodes() []domain.NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var all []domain.NodeRecord
	for _, node := range s.nodes {
		all = append(all, node)
	}
	return all
}
