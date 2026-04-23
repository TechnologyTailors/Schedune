use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub enum HealthState {
    Healthy,
    Warning,
    Degraded,
    Unschedulable,
    Quarantined,
    Unknown,
}

#[derive(Debug, Serialize, Deserialize)]
pub enum AlarmSeverity {
    Info,
    Warning,
    Error,
    Critical,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct ActiveAlarm {
    pub source: String,
    pub severity: AlarmSeverity,
    pub code: String,
    pub description: String,
    pub remediation_hint: Option<String>,
    pub timestamp_sec: u64,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct NodeHealth {
    pub state: HealthState,
    pub active_alarms: Vec<ActiveAlarm>,
}
