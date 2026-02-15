package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
	colorReset  = "\033[0m"
)

func logInfof(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, colorBlue+"[orchestrator] "+colorReset+format+"\n", args...)
}

func logOKf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, colorGreen+"✓ "+colorReset+format+"\n", args...)
}

func logWarnf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, colorYellow+"⚠ "+colorReset+format+"\n", args...)
}

func logErrf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, colorRed+"✗ "+colorReset+format+"\n", args...)
}

// DBService describes how to start and health check a database container.
type DBService struct {
	Name       string
	Service    string   // docker-compose service name
	ReadyCheck []string // command to verify readiness (passed to docker exec)
}

// DefaultServices returns the standard list of databases in benchmark order.
func DefaultServices() []DBService {
	return []DBService{
		{
			Name:       "postgres",
			Service:    "postgres",
			ReadyCheck: []string{"docker", "exec", "benchmark-postgres", "pg_isready", "-U", "benchmark"},
		},
		{
			Name:       "mongodb",
			Service:    "mongodb",
			ReadyCheck: []string{"docker", "exec", "benchmark-mongodb", "mongosh", "--quiet", "--eval", "db.adminCommand('ping').ok"},
		},
		{
			Name:       "clickhouse",
			Service:    "clickhouse",
			ReadyCheck: []string{"docker", "exec", "benchmark-clickhouse", "clickhouse-client", "--query", "SELECT 1"},
		},
		{
			Name:       "cassandra",
			Service:    "cassandra",
			ReadyCheck: []string{"docker", "exec", "benchmark-cassandra", "cqlsh", "-e", "DESCRIBE KEYSPACES"},
		},
	}
}

// ServiceByName returns the DBService for a given database name.
func ServiceByName(name string) (DBService, bool) {
	for _, s := range DefaultServices() {
		if s.Name == name {
			return s, true
		}
	}

	return DBService{}, false
}

// StartService brings up a docker-compose service.
func StartService(ctx context.Context, service string) error {
	logInfof("Starting %s...", service)

	cmd := exec.CommandContext(ctx, "docker-compose", "up", "-d", service)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

// StopService stops and removes a docker-compose service.
func StopService(ctx context.Context, service string) error {
	logWarnf("Stopping %s to free memory...", service)

	stop := exec.CommandContext(ctx, "docker-compose", "stop", service)

	err := stop.Run()
	if err != nil {
		logErrf("%v", err)
	}

	rm := exec.CommandContext(ctx, "docker-compose", "rm", "-f", service)

	return rm.Run()
}

// WaitReady polls the readiness check until it succeeds or the context is canceled.
func WaitReady(ctx context.Context, svc DBService) error {
	logInfof("Waiting for %s to be ready...", svc.Name)

	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return ctx.Err()
	}

	deadline := time.After(60 * time.Second)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			logErrf("%s: readiness timeout after 60s", svc.Name)
			return fmt.Errorf("%s: readiness timeout after 60s", svc.Name)
		case <-ticker.C:
			if runReadyCheck(ctx, svc.ReadyCheck) == nil {
				logOKf("%s is ready", svc.Name)
				return nil
			}
		}
	}
}

// runReadyCheck executes a readiness check command.
// The commands are defined internally in DefaultServices, not from user input.
func runReadyCheck(ctx context.Context, args []string) error {
	return exec.CommandContext(ctx, args[0], args[1:]...).Run()
}

// Cleanup tears down all docker-compose services and removes volumes.
func Cleanup(ctx context.Context) error {
	logWarnf("Cleaning up containers and volumes...")

	cmd := exec.CommandContext(ctx, "docker-compose", "down", "-v")

	if err := cmd.Run(); err != nil {
		logErrf("Cleanup failed: %v", err)
		return err
	}

	logOKf("Cleanup complete")

	return nil
}
