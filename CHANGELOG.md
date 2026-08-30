# Changelog

## Release Notes: penguins-tailor v26.8.30 - 2026-08-30
This release introduces real-time target name feedback during costume and accessory execution, refines terminology to "system configuration" (sysroot), standardizes all user-facing console output and internal logs into English (i18n), optimizes the package management pipeline by removing redundant batch chunking, polishes the split-screen TUI header to a concise 3-line layout, expands automated CI testing across multi-architecture distributions ("Hammers" workflow), and introduces a comprehensive unit testing suite.

### 🎯 Real-Time Execution Feedback & Terminology Refinement
* **Active Target Name in Progress Messages**: Updated split-screen action messages (`pkg/tailor/wear.go`) to display the active costume or accessory name (e.g., `"<target>, installing packages..."`, `"<target>, running finalization scripts..."`), providing granular visibility during multi-stage runs.
* **System Configuration Terminology**: Renamed "sysroot overlay" to "system configuration (sysroot)" across CLI actions, log messages, and documentation (`README.md`, `pkg/tailor/wear.go`) to more accurately reflect filesystem configuration syncing.
* **Accessory Step Log Isolation**: Prevented subsidiary accessory installation steps from polluting the primary costume's high-level step log in split-screen mode.

### 🌐 Full English Internationalization (i18n) & CLI Polish
* **Complete English Translation**: Translated all remaining Italian console output, terminal prints, summary messages, warnings, execution logs, and code comments to English across `cmd/wear.go`, `pkg/tailor/report.go`, `pkg/tailor/repositories.go`, `pkg/tailor/tailor.go`, `pkg/tailor/types.go`, `pkg/tailor/wear-logic.go`, `pkg/tailor/wear.go`, and `pkg/utils/`.
* **Atelier Repository Terminology**: Updated `tailor get` help and flag descriptions (`cmd/get.go`) to clarify repository terminology and wardrobe destination (`~/.wardrobe`).
* **TUI Header & Title Layout**: Cleaned up the split-screen TUI appearance (`pkg/utils/split_screen.go`, `pkg/tailor/wear.go`) by standardizing on a clean 3-line header and removing decorative icons for a polished CLI experience.

### ⚡ Package Pipeline Optimization & Dry-Run Enhancements
* **Direct Package Installation**: Removed package batching and chunking routines (`processPackagesInBatches` in `pkg/tailor/wear-logic.go`), simplifying the package installation pipeline and reducing execution overhead.
* **Dry-Run Simulation Parity**: Enhanced `--dry-run` simulation mode to accurately simulate interactive package configurations, system configuration overlays, and finalization scripts without requiring root permissions.

### 🔨 Multi-Arch CI Matrix ("Hammers") & Test Coverage
* **Multi-Distribution Packaging CI**: Added GitHub Actions workflow (`.github/workflows/hammers.yml`) to compile and package native distributions across Debian (amd64, arm64, riscv64), Ubuntu, Fedora, openSUSE, Alpine, Arch Linux, and Manjaro Linux.
* **Automated Unit Test Suites**: Added unit testing coverage in `pkg/tailor/wear_test.go`, `cmd/wear_test.go`, and `pkg/utils/split_screen_test.go` to validate costume dry-runs, accessory modes, TUI header rendering, and report generation.

## Release Notes: penguins-tailor v26.8.28 - 2026-08-28
This release introduces automatic discovery of `packages.yaml` configurations across costume directories, refactors the costume loading pipeline, removes legacy reconcile logic, and refines wardrobe repository synchronization.

### 📦 Dynamic Package Discovery & Architecture Refactoring
* **Automatic packages.yaml Discovery**: Added recursive discovery for `packages.yaml` across costume subdirectories, automatically aggregating distribution-specific package definitions.
* **Legacy Reconcile Removal**: Removed outdated `reconcile.go` logic to streamline costume staging and execution.
* **Wardrobe Integration**: Improved repository and keyrings configuration handling when applying costumes from wardrobe ateliers.
