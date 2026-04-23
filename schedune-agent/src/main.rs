mod api;
mod capabilities;
mod collectors;
mod config;
mod daemon;
mod health;
mod migration;
mod scheduler_contract;

use anyhow::Result;
use clap::{Parser, Subcommand};

use collectors::system::SystemCollector;
use migration::MigrationAnalyzer;
use scheduler_contract::SchedulerEnvelope;

/// Schedune Node Agent - ARM-Native Infrastructure Control Plane
#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Discovers facts, capabilities, and constraints to produce a Scheduler Envelope
    Inspect,
    /// Assesses a workload image and produces a Readiness Report
    Analyze {
        /// Path to the VMDK, QCOW2, or OVA to analyze
        #[arg(short, long)]
        image_path: String,
    },
    /// Starts the thin runtime supervisor
    Serve {
        /// Port to listen on
        #[arg(short, long, default_value_t = 8080)]
        port: u16,
    },
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();

    let cli = Cli::parse();

    match &cli.command {
        Commands::Inspect => {
            let (compatibility, facts, mut capabilities, mut constraints, health, sys_status) = SystemCollector::collect();
            let (mut virt_caps, mut virt_constraints, virt_status) = collectors::virtualization::VirtualizationCollector::collect();
            
            capabilities.append(&mut virt_caps);
            constraints.append(&mut virt_constraints);

            // The agent emits TRUTH via the rigid Envelope contract.
            // It does not decide WHERE workloads go. It just reports its state.
            let node_id = facts.os.hostname.clone();
            
            let envelope = SchedulerEnvelope::new(
                node_id,
                compatibility,
                facts,
                capabilities,
                constraints,
                health,
                vec![sys_status, virt_status],
            );

            let json = serde_json::to_string_pretty(&envelope)?;
            println!("{}", json);
        }
        Commands::Analyze { image_path } => {
            let analyzer = MigrationAnalyzer::new(image_path);
            let report = analyzer.analyze();
            let json = serde_json::to_string_pretty(&report)?;
            println!("{}", json);
        }
        Commands::Serve { port } => {
            daemon::run(*port).await;
        }
    }

    Ok(())
}

#[cfg(test)]
mod fixtures_test;
