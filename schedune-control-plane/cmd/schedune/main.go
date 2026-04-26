package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
	INFO = "[\033[36mINFO\033[0m]"
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

func checkCommand(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

func getKernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "unknown"
	}
	return string(out[:len(out)-1])
}

func checkPort(port string) bool {
	l, err := net.Listen("tcp", port)
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func getBinaryVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		return "unknown"
	}

	version := strings.TrimSpace(lines[0])
	if len(version) > 128 {
		version = version[:128]
	}
	if version == "" {
		return "unknown"
	}
	return version
}

func runDoctor() {
	fmt.Println("============================================")
	fmt.Println("              Schedune Doctor               ")
	fmt.Println("============================================")
	fmt.Println()

	var vmLaunchReady, chReady, fcReady bool
	var kvmExists bool
	var controlPlaneReady = true
	var agentInspectReady = true
	var orphanSweepReady = true

	fmt.Println("Host")
	fmt.Println("----")
	if runtime.GOOS == "linux" {
		fmt.Printf("%s OS: %s\n", PASS, runtime.GOOS)
		fmt.Printf("%s Architecture: %s\n", PASS, runtime.GOARCH)
		fmt.Printf("%s Kernel: %s\n", PASS, getKernelVersion())
	} else {
		fmt.Printf("%s OS: %s (Control plane: evaluator mode. Agent/Runtime: Linux required)\n", WARN, runtime.GOOS)
		agentInspectReady = false
	}

	if syscall.Access("/proc", 4) == nil {
		fmt.Printf("%s /proc: readable\n", PASS)
	} else {
		fmt.Printf("%s /proc: not readable\n", WARN)
		orphanSweepReady = false
	}
	fmt.Println()

	fmt.Println("Storage / Paths")
	fmt.Println("---------------")
	dbDir := "./var"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		fmt.Printf("%s SQLite DB Path: %s not writable (%v)\n", FAIL, dbDir, err)
		controlPlaneReady = false
	} else if syscall.Access(dbDir, 2) == nil {
		fmt.Printf("%s SQLite DB Path: %s/schedune.db (writable)\n", PASS, dbDir)
	} else {
		fmt.Printf("%s SQLite DB Path: %s not writable\n", FAIL, dbDir)
		controlPlaneReady = false
	}

	runDir := "./var/run"
	if err := os.MkdirAll(runDir, 0755); err == nil && syscall.Access(runDir, 2) == nil {
		fmt.Printf("%s Runtime Path: %s (writable)\n", PASS, runDir)
	} else {
		fmt.Printf("%s Runtime Path: %s not writable\n", FAIL, runDir)
	}

	fmt.Printf("%s Temp Path: %s (writable)\n", PASS, os.TempDir())
	fmt.Println()

	fmt.Println("Virtualization Primitives")
	fmt.Println("-------------------------")
	if _, err := os.Stat("/dev/kvm"); err == nil {
		kvmExists = true
		if syscall.Access("/dev/kvm", 6) == nil {
			fmt.Printf("%s /dev/kvm: present and openable\n", PASS)
			vmLaunchReady = true
		} else {
			fmt.Printf("%s /dev/kvm: present but not openable by current user\n", WARN)
		}
	} else {
		fmt.Printf("%s /dev/kvm: missing\n", INFO)
	}

	if _, err := os.Stat("/dev/net/tun"); err == nil {
		fmt.Printf("%s /dev/net/tun: present\n", PASS)
	} else {
		fmt.Printf("%s /dev/net/tun: missing\n", INFO)
	}

	if syscall.Access("/sys/fs/cgroup", 4) == nil {
		fmt.Printf("%s cgroups: readable\n", PASS)
	} else {
		fmt.Printf("%s cgroups: not readable\n", INFO)
	}
	fmt.Println()

	fmt.Println("Backend Binaries")
	fmt.Println("----------------")
	qemuBin := "qemu-system-x86_64"
	if runtime.GOARCH == "arm64" {
		qemuBin = "qemu-system-aarch64"
	}
	if path := checkCommand(qemuBin); path != "" {
		fmt.Printf("%s QEMU: %s\n", PASS, path)
	} else {
		fmt.Printf("%s QEMU: %s not found\n", INFO, qemuBin)
		vmLaunchReady = false
	}

	if path := checkCommand("cloud-hypervisor"); path != "" {
		fmt.Printf("%s Cloud Hypervisor: %s\n", PASS, path)
		chReady = true
	} else {
		fmt.Printf("%s Cloud Hypervisor: not found\n", INFO)
	}

	if path := checkCommand("firecracker"); path != "" {
		fmt.Printf("%s Firecracker: %s\n", PASS, path)
		fcReady = true
	} else {
		fmt.Printf("%s Firecracker: not found\n", INFO)
	}
	fmt.Println()

	fmt.Println("Control Plane")
	fmt.Println("-------------")
	if checkPort("127.0.0.1:9090") {
		fmt.Printf("%s API Port: 127.0.0.1:9090 is available\n", PASS)
	} else {
		fmt.Printf("%s API Port: 127.0.0.1:9090 is in use or unavailable\n", FAIL)
		controlPlaneReady = false
	}
	if controlPlaneReady {
		fmt.Printf("%s SQLite Initialization: possible\n", PASS)
	} else {
		fmt.Printf("%s SQLite Initialization: blocked by path errors\n", FAIL)
	}
	fmt.Println()

	fmt.Println("Summary")
	fmt.Println("-------")
	if controlPlaneReady {
		fmt.Printf("%s Control plane\n", PASS)
	} else {
		fmt.Printf("%s Control plane\n", FAIL)
	}

	if agentInspectReady {
		fmt.Printf("%s Agent inspect\n", PASS)
	} else {
		fmt.Printf("%s Agent inspect\n", FAIL)
	}

	if vmLaunchReady {
		fmt.Printf("%s VM launch\n", PASS)
	} else if kvmExists {
		fmt.Printf("%s VM launch (missing QEMU or permissions)\n", WARN)
	} else {
		fmt.Printf("%s VM launch (missing /dev/kvm)\n", INFO)
	}

	if chReady && kvmExists {
		fmt.Printf("%s Cloud Hypervisor support\n", PASS)
	} else {
		fmt.Printf("%s Cloud Hypervisor support (missing binary or /dev/kvm)\n", INFO)
	}

	if fcReady && kvmExists {
		fmt.Printf("%s Firecracker support\n", PASS)
	} else {
		fmt.Printf("%s Firecracker support (missing binary or /dev/kvm)\n", INFO)
	}

	if orphanSweepReady {
		fmt.Printf("%s Orphan sweep prerequisites\n", PASS)
	} else {
		fmt.Printf("%s Orphan sweep prerequisites\n", FAIL)
	}
	fmt.Println()

	fmt.Println("Recommended next step:")
	if runtime.GOOS != "linux" && controlPlaneReady {
		fmt.Println("  make demo-fixture (evaluator mode)")
	} else if controlPlaneReady && agentInspectReady {
		fmt.Println("  make demo")
	} else {
		fmt.Println("  Resolve [FAIL] items before continuing.")
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
	enumerator := cp_runtime.NewEnumerator()
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
	systemHandler := api.NewSystemHandler()
	nodeHandler := api.NewNodeHandler(memStore)
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
		// System Health
		v1.GET("/healthz", systemHandler.Healthz)

		// Operator -> Control Plane (Nodes)
		v1.GET("/nodes", nodeHandler.ListNodes)
		v1.GET("/nodes/:id", nodeHandler.GetNode)

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
