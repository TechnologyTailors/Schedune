package domain

import (
	"sort"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/pkg/schema/workload"
	"time"
)

type ScoredCandidate struct {
	NodeID      string            `json:"node_id"`
	Score       int               `json:"score"`
	Eligibility EligibilityResult `json:"eligibility"`
}

type ScheduleDecision struct {
	WorkloadID    string              `json:"workload_id"`
	SelectedNode  *string             `json:"selected_node"`
	RankedNodes   []ScoredCandidate   `json:"ranked_nodes"`
	RejectedNodes []EligibilityResult `json:"rejected_nodes"`
}

// ScheduleV0 filters candidates via Evaluate() and scores the eligible ones.
func ScheduleV0(intent workload.WorkloadIntent, candidateNodes []NodeRecord) ScheduleDecision {
	decision := ScheduleDecision{
		WorkloadID:    intent.WorkloadID,
		RankedNodes:   []ScoredCandidate{},
		RejectedNodes: []EligibilityResult{},
	}

	now := time.Now().Unix()

	for _, node := range candidateNodes {
		result := Evaluate(intent, node)

		if !result.IsEligible {
			decision.RejectedNodes = append(decision.RejectedNodes, result)
			continue
		}

		// --- Scoring v0 (Keep it boring and obvious) ---
		score := 0

		// +30 for extremely fresh telemetry (< 30 seconds)
		if now-node.Freshness.LastCollectionTime.Unix() < 30 {
			score += 30
		}

		// +20 if it has zero active alarms
		if len(node.Health.ActiveAlarms) == 0 {
			score += 20
		}

		// +10 for every soft preference matched
		score += len(result.MatchedFeatures) * 10

		decision.RankedNodes = append(decision.RankedNodes, ScoredCandidate{
			NodeID:      node.ID,
			Score:       score,
			Eligibility: result,
		})
	}

	// Sort RankedNodes descending by Score
	sort.Slice(decision.RankedNodes, func(i, j int) bool {
		return decision.RankedNodes[i].Score > decision.RankedNodes[j].Score
	})

	if len(decision.RankedNodes) > 0 {
		selected := decision.RankedNodes[0].NodeID
		decision.SelectedNode = &selected
	}

	return decision
}
