use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub enum MigrationScore {
    Green,
    Yellow,
    Red,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct MigrationBlocker {
    pub severity: String,
    pub description: String,
    pub remediation: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct MigrationReadinessReport {
    pub schema_version: String,
    pub image_path: String,
    pub detected_architecture: String,
    pub detected_os_family: String,
    pub score: MigrationScore,
    pub blockers: Vec<MigrationBlocker>,
}

pub struct MigrationAnalyzer {
    pub image_path: String,
}

impl MigrationAnalyzer {
    pub fn new(image_path: &str) -> Self {
        Self {
            image_path: image_path.to_string(),
        }
    }

    pub fn analyze(&self) -> MigrationReadinessReport {
        tracing::info!("Analyzing image: {}", self.image_path);
        
        let is_vmdk = self.image_path.ends_with(".vmdk");
        let mut blockers = Vec::new();
        let mut score = MigrationScore::Green;

        if is_vmdk {
            score = MigrationScore::Yellow;
            blockers.push(MigrationBlocker {
                severity: "High".to_string(),
                description: "Image is a VMware VMDK file. Requires format conversion.".to_string(),
                remediation: Some("Convert to QCOW2 and inject qemu-guest-agent.".to_string()),
            });
        }

        blockers.push(MigrationBlocker {
            severity: "Medium".to_string(),
            description: "x86 Architecture Detected.".to_string(),
            remediation: Some("Schedule in x86 holding pool until application recompilation.".to_string()),
        });
        
        if score == MigrationScore::Green {
            score = MigrationScore::Yellow;
        }

        MigrationReadinessReport {
            schema_version: "v1alpha1".to_string(),
            image_path: self.image_path.clone(),
            detected_architecture: "x86_64".to_string(),
            detected_os_family: "linux".to_string(),
            score,
            blockers,
        }
    }
}
