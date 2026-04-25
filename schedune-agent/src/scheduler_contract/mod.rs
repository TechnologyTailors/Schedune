use crate::capabilities::{CompatibilityClassification, NodeCapability, NodeConstraint, NodeFacts};
use crate::health::NodeHealth;
use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
pub struct CollectorStatus {
    pub collector_name: String,
    pub success: bool,
    pub duration_ms: u64,
    pub error_message: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SchedulerEnvelope {
    pub schema_version: String,
    pub agent_version: String,
    pub collection_id: String,
    pub timestamp_sec: u64,
    pub node_id: String,

    pub compatibility: CompatibilityClassification,
    pub facts: NodeFacts,
    pub capabilities: Vec<NodeCapability>,
    pub constraints: Vec<NodeConstraint>,
    pub health: NodeHealth,

    pub collector_statuses: Vec<CollectorStatus>,
}

impl SchedulerEnvelope {
    pub fn new(
        node_id: String,
        compatibility: CompatibilityClassification,
        facts: NodeFacts,
        capabilities: Vec<NodeCapability>,
        constraints: Vec<NodeConstraint>,
        health: NodeHealth,
        collector_statuses: Vec<CollectorStatus>,
    ) -> Self {
        let timestamp_sec = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();

        Self {
            schema_version: "v1alpha1".to_string(),
            agent_version: env!("CARGO_PKG_VERSION").to_string(),
            collection_id: Uuid::new_v4().to_string(),
            timestamp_sec,
            node_id,
            compatibility,
            facts,
            capabilities,
            constraints,
            health,
            collector_statuses,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::capabilities::{
        CompatibilityClassType, CpuFacts, MemoryFacts, OsFacts, Provenance, SupportState,
    };
    use crate::health::HealthState;

    #[test]
    fn test_golden_payload_healthy_arm_production() {
        let facts = NodeFacts {
            cpu: CpuFacts {
                architecture: "aarch64".to_string(),
                cores: 128,
                vendor_id: Some("ARM".to_string()),
            },
            memory: MemoryFacts { total_mb: 262144 },
            os: OsFacts {
                hostname: "arm-node-01".to_string(),
                name: "Ubuntu".to_string(),
                kernel_version: Some("6.8.0-110-generic".to_string()),
            },
        };

        let class = CompatibilityClassification {
            class: CompatibilityClassType::ArmProduction,
            reason_codes: vec!["CLASS_ARM_PROD_READY".to_string()],
        };

        let capabilities = vec![
            NodeCapability {
                feature: "kvm_virtualization".to_string(),
                state: SupportState::Supported,
                provenance: Provenance::Observed,
                reason_code: Some("CAP_KVM_SUPPORTED".to_string()),
                version: None,
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            },
            NodeCapability {
                feature: "qemu_binary_present".to_string(),
                state: SupportState::Supported,
                provenance: Provenance::Observed,
                reason_code: Some("CAP_QEMU_BINARY_PRESENT".to_string()),
                version: None,
                observed_at_sec: 1776978000,
                stale_after_sec: Some(1776978300),
            },
        ];

        let health = NodeHealth {
            state: HealthState::Healthy,
            active_alarms: vec![],
        };

        let status = CollectorStatus {
            collector_name: "MockCollector".to_string(),
            success: true,
            duration_ms: 10,
            error_message: None,
        };

        let envelope = SchedulerEnvelope::new(
            "arm-node-01".to_string(),
            class,
            facts,
            capabilities,
            vec![], // No constraints
            health,
            vec![status],
        );

        let json = serde_json::to_string(&envelope).unwrap();
        assert!(json.contains("v1alpha1"));
        assert!(json.contains("CLASS_ARM_PROD_READY"));
        assert!(json.contains("ArmProduction"));
        assert!(json.contains("aarch64"));
        assert!(json.contains("arm-node-01"));
    }
}
