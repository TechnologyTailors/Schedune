package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/api"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/recovery"
	cp_runtime "github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/runtime/inspect"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store"
	"github.com/TechnologyTailors/Schedune/schedune-control-plane/internal/store/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	PASS = "[\033[32mOK\033[0m]"
	WARN = "[\033[33mWARN\033[0m]"
	FAIL = "[\033[31mFAIL\033[0m]"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "server":
		runServer()
	case "doctor":
		runDoctor()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Schedune Control Plane")
	fmt.Println("\nUsage:")
	fmt.Println("  schedune <command>")
	fmt.Println("\nCommands:")
	fmt.Println("  server    Start the control plane API server")
	fmt.Println("  doctor    Run the preflight checks to verify local readiness")
}

func checkCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runDoctor() {
	fmt.Println("============================================")
	fmt.Println("    Schedune Preflight & Doctor Check       ")
	fmt.Println("============================================")

	vmLaunchReady := true
	chReady := true
	fcReady := true

	// 1. OS Check
	if runtime.GOOS == "linux" {
		fmt.Printf("%s Linux host detected: %s\n", PASS, runtime.GOARCH)
	} else {
		fmt.Printf("%s Schedune requires Linux. Found: %s\n", FAIL, runtime.GOOS)
		vmLaunchReady = false
		chReady = false
		fcReady = false
	}

	// 2. SQLite / Persistence Check
	dbDir := "./var"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		fmt.Printf("%s Database directory not writable: %s (%v)\n", FAIL, dbDir, err)
	} else if syscall.Access(dbDir, 2) == nil {
		fmt.Printf("%s SQLite writable: %s/schedune.db\n", PASS, dbDir)
	} else {
		fmt.Printf("%s Database directory not writable: %s\n", FAIL, dbDir)
	}

	// 3. KVM Check
	kvmExists := false
	if _, err := os.Stat("/dev/kvm"); err == nil {
		kvmExists = true
		if syscall.Access("/dev/kvm", 6) == nil {
			fmt.Printf("%s /dev/kvm is present and writable\n", PASS)
		} else {
			fmt.Printf("%s /dev/kvm exists but is not writable by current user. VM launch will fail.\n", WARN)
			vmLaunchReady = false
			chReady = false
			fcReady = false
		}
	} else {
		fmt.Printf("%s /dev/kvm missing: VM execution will not work on this host.\n", WARN)
		vmLaunchReady = false
		chReady = false
		fcReady = false
	}

	// 4. QEMU Binary Check
	qemuBin := "qemu-system-x86_64"
	if runtime.GOARCH == "arm64" {
		qemuBin = "qemu-system-aarch64"
	}
	if checkCommand(qemuBin) {
		fmt.Printf("%s QEMU binary found: %s\n", PASS, qemuBin)
	} else {
		fmt.Printf("%s %s not found in PATH. KVM_QEMU backend will fail.\n", WARN, qemuBin)
		vmLaunchReady = false
	}

	// 5. Cloud Hypervisor Binary Check
	if checkCommand("cloud-hypervisor") {
		fmt.Printf("%s Cloud Hypervisor binary found\n", PASS)
	} else {
		fmt.Printf("%s cloud-hypervisor not found in PATH. CLOUD_HYPERVISOR backend will fail.\n", WARN)
		chReady = false
	}

	// 6. Firecracker Checks
	if checkCommand("firecracker") {
		fmt.Printf("%s Firecracker binary found\n", PASS)
	} else {
		fmt.Printf("%s firecracker not found in PATH. MicroVM launch validation will fail.\n", WARN)
		fcReady = false
	}

	if _, err := os.Stat("/dev/net/tun"); err == nil {
		fmt.Printf("%s /dev/net/tun is present (required for Firecracker networking)\n", PASS)
	} else {
		fmt.Printf("%s /dev/net/tun missing: Firecracker networking checks may fail.\n", WARN)
		fcReady = false
	}

	// 7. ProcFS
	if syscall.Access("/proc", 4) == nil {
		fmt.Printf("%s /proc is readable (required for orphan sweep)\n", PASS)
	} else {
		fmt.Printf("%s /proc is not readable. Orphan detection will fail.\n", FAIL)
	}

	fmt.Println("\nSchedune doctor summary:")
	fmt.Println("- Control plane: runnable")
	fmt.Println("- Agent inspect: runnable")
	if vmLaunchReady {
		fmt.Println("- VM launch: ready")
	} else if !kvmExists {
		fmt.Println("- VM launch: not ready (missing /dev/kvm)")
	} else {
		fmt.Println("- VM launch: partial")
	}

	if chReady {
		fmt.Println("- Cloud Hypervisor readiness: ready")
	} else {
		fmt.Println("- Cloud Hypervisor readiness: partial")
	}

	if fcReady {
		fmt.Println("- Firecracker readiness: ready")
	} else {
		fmt.Println("- Firecracker readiness: partial")
	}
}

func runServer() {
	gin.SetMode(gin.ReleaseMode)

	// Initialize the durable SQLite store
	os.MkdirAll("./var", 0755)
	dbPath := "./var/schedune.db"
	sqliteStore, err := sqlite.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize SQLite store")
	}

	// Phase 6: Bootstrap Recovery
	bootstrapper := recovery.NewRecoveryBootstrapper(sqliteStore, sqliteStore, &inspect.ProcessInspector{})
	if err := bootstrapper.Bootstrap(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to run recovery bootstrap")
	}

	// Phase 7B.5: Real Orphan Sweep
	enumerator := &cp_runtime.LinuxProcEnumerator{}
	orphanSweeper := recovery.NewOrphanSweepService(enumerator, sqliteStore, sqliteStore, sqliteStore, "local-node")

	go func() {
		for {
			time.Sleep(30 * time.Second)
			if err := orphanSweeper.SweepOnce(context.Background()); err != nil {
				log.Error().Err(err).Msg("Orphan sweep failed")
			}
		}
	}()

	// We still use InMemoryStore for some node truth logic in V0/V1 that wasn't moved to SQLite yet
	memStore := store.NewInMemoryStore()

	// Initialize HTTP Handlers
	intakeHandler := api.NewIntakeHandler(memStore)
	schedulerHandler := api.NewSchedulerHandler(memStore)
	launchHandler := api.NewLaunchHandler(memStore, sqliteStore)
	orphanHandler := api.NewOrphanHandler(sqliteStore)

	// Setup Router
	r := gin.New()
	r.Use(gin.Recovery())

	// API Group
	v1 := r.Group("/api/v1alpha1")
	{
		// Data Plane -> Control Plane
		v1.POST("/intake/envelope", intakeHandler.Ingest)

		// Operator -> Control Plane (Explainability)
		v1.GET("/nodes/:id/explain", intakeHandler.ExplainNodeDecision)

		// Workload Orchestration -> Control Plane
		v1.POST("/schedule/explain", schedulerHandler.ExplainSchedule)
		v1.POST("/schedule/select", schedulerHandler.SelectNode)

		// Execution Contract -> Control Plane
		v1.POST("/launch/validate", launchHandler.ValidateLaunch)
		v1.POST("/launch/dry-run", launchHandler.DryRunLaunch)
		v1.POST("/launch/execute", launchHandler.ExecuteLaunch)
		v1.GET("/launch/:id", launchHandler.InspectLaunch)
		v1.GET("/launch/:id/readiness", launchHandler.InspectReadiness)
		v1.GET("/launch/:id/trace", launchHandler.InspectTrace)
		v1.GET("/launch/:id/events", launchHandler.InspectEvents)
		v1.POST("/launch/:id/terminate", launchHandler.TerminateLaunch)

		// Recovery -> Control Plane
		v1.GET("/recovery/orphans", orphanHandler.ListOrphans)
		v1.GET("/recovery/orphans/:id", orphanHandler.GetOrphan)
	}

	log.Info().
		Str("API", "http://127.0.0.1:9090").
		Str("SQLite", dbPath).
		Str("RecoveryEpoch", bootstrapper.Epoch).
		Msg("Schedune control plane starting...")

	if err := r.Run(":9090"); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
