# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`ph` (Process Hunter) is a parental control tool that monitors OS processes and terminates those that exceed specified time limits or run during downtime periods. It's designed to limit game time for children by monitoring processes and enforcing daily time budgets and blackout periods.

## Architecture

The codebase is structured into several key packages:

- **`engine/`** - Core process monitoring and killing logic
  - `ProcessHunter` struct manages configuration, time balance tracking, and process termination
  - Supports day-specific time limits and downtime periods via JSON configuration
  - Uses goroutines with context cancellation for graceful shutdown
  - Checks running processes every 3 minutes (hardcoded)

- **`server/`** - HTTP web interface and API
  - Serves static web UI at localhost:8080
  - Provides REST endpoints for configuration and process balance data
  - Basic auth protection for PUT operations (username: "time", password: "k33p3rs")
  - Uses embedded files for web assets

- **`cmd/cli/`** - Main CLI application entry point
- **`cmd/winsvc/`** - Windows service wrapper

## Development Commands

### Building
```bash
make build          # Build CLI and Windows service binaries
make build_test     # Build with test processes
make clean          # Clean build artifacts
```

### Testing
```bash
make test           # Run tests with coverage
make cover          # Generate HTML coverage report
```

### Running
```bash
make run            # Build and run with test processes
# OR run directly:
cd bin && ./ph      # On Unix
cd bin && ph.exe    # On Windows
```

For Windows service development:
```bash
phsvc debug         # Run as CLI tool (not as service)
phsvc install       # Install as Windows service
phsvc start         # Start service
phsvc stop          # Stop service
phsvc remove        # Uninstall service
```

## Configuration

The application uses `cfg.json` for process limits and downtime configuration. The configuration format supports:
- Process groups with shared time budgets
- Day-specific limits (weekdays, dates, wildcards)
- Downtime periods in HH:MM..HH:MM format
- Priority-based rule matching (specific dates > day lists > individual days > wildcards)

## Key Files

- `engine/ph.go` - Main ProcessHunter implementation with time tracking and process killing logic
- `server/server.go` - Web server with API endpoints and basic auth
- `cmd/cli/main.go` - CLI entry point with graceful shutdown handling
- `Makefile` - Cross-platform build system
- `testdata/cfg.json` - Example configuration file

## Testing Notes

The project includes test processes (`test_process/`) that can be built and run to simulate monitored applications during development and testing.

## Web Interface

The web UI is available at http://localhost:8080 and provides:
- Real-time process balance monitoring
- Configuration editing (with basic auth)
- Process group status display