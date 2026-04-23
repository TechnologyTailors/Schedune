use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub enum CompatibilityClassType {
    ArmProduction,
    X86HoldingPool,
    Unsupported,
    Degraded,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CompatibilityClassification {
    pub class: CompatibilityClassType,
    pub reason_codes: Vec<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct CpuFacts {
    pub architecture: String,
    pub cores: u32,
    pub vendor_id: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct MemoryFacts {
    pub total_mb: u64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct OsFacts {
    pub hostname: String,
    pub name: String,
    pub kernel_version: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct NodeFacts {
    pub cpu: CpuFacts,
    pub memory: MemoryFacts,
    pub os: OsFacts,
}

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub enum SupportState {
    Supported,
    Unsupported,
    Unknown,
    Unavailable,
}

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub enum Provenance {
    Observed,
    Inferred,
    Unavailable(String),
}

#[derive(Debug, Serialize, Deserialize)]
pub struct NodeCapability {
    pub feature: String,
    pub state: SupportState,
    pub provenance: Provenance,
    pub reason_code: Option<String>,
    pub observed_at_sec: u64,
    pub stale_after_sec: Option<u64>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct NodeConstraint {
    pub constraint_type: String,
    pub code: String,
    pub description: String,
    pub observed_value: Option<String>,
    pub expected_value: Option<String>,
}
