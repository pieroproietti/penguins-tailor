# Changelog

## Release Notes: penguins-tailor v26.9.2 - 2026-09-02
This release strengthens the `tailor wear` package-management lifecycle, replaces the split-screen dashboard with clearer progressive terminal output, centralizes technical logging and detailed reports, and makes package results and repository-dependent kernel header handling more reliable.

### 📦 Package Management & Distribution Safety
* **Package Manager Abstraction**: Introduced a dedicated `PackageManager` interface and moved APT-specific refresh, upgrade, installation, healing, and installed-package checks into a separate implementation (`pkg/tailor/package-manager.go`, `pkg/tailor/package-manager-apt.go`).
* **Distribution Validation**: Added explicit distribution-family checks before applying a costume and removed the obsolete AI prompt path from the wear pipeline.
* **Portable Repository Upgrades**: Repository setup now delegates refresh and upgrade operations to the active package manager instead of invoking APT-specific upgrade logic directly.
* **Early Metadata Refresh**: Package metadata is refreshed before wardrobe and costume setup, and a failed initial refresh now stops the operation before any package or kernel-header work begins.

### 🔄 Repository & Kernel Header Ordering
* **Repository-Aware DKMS Preparation**: Moved the running-kernel header check after costume repository configuration and metadata updates, while keeping it before package installation so newly configured sources can provide the required header packages.
* **Single Main-Costume Check**: Kernel headers are verified once for the main costume rather than repeatedly for each accessory.
* **Lifecycle Test Coverage**: Added tests covering metadata refresh failure, repository refresh, kernel-header verification, and package-installation ordering.

### 🖥️ Progressive Wear Output
* **Progressive Terminal Renderer**: Replaced the fixed split-screen dashboard with a progressive action stream that preserves completed steps and keeps the current action visibly updated.
* **Clearer Costume Progress**: Added the main costume to the completed-step history, improved package counters, and removed the redundant top header divider.
* **Report Review Flow**: The complete wear output remains visible before the interactive reset prompt and final summary, making warnings and command results easier to review.

### 📝 Logging, Reports & Package Outcomes
* **Central Technical Log**: Added a concurrency-safe structured logger and raw command-output capture under `/var/log/tailor/tailor.log`, consolidating operational diagnostics in one location (`pkg/utils/technical_logger.go`, `pkg/utils/exec.go`).
* **Unavailable vs. Failed Packages**: Package results now distinguish packages absent from configured repositories from packages that were available but failed during installation.
* **Detailed Wear Reports**: Timestamped reports under `/var/log/tailor/` now include separate Installed, Unavailable, and Could NOT be installed sections, with the final summary reporting the relevant totals and report path.
* **Expanded Regression Tests**: Added coverage for package-manager behavior, technical logging, report contents, progressive output, and package-result aggregation.

## Release Notes: penguins-tailor v26.8.31 - 2026-08-31
This release introduces the `tailor tools repo` command to manage official penguins-eggs repositories across diverse Linux distributions, refactors the execution lifecycle to defer costume finalization after accessory processing, introduces explicit distinction between sequence and finalization commands, adds an interactive terminal pause before displaying summary reports, and enables automatic reboot execution when required by a costume.

### 🐧 Official Repository Management (`tailor tools repo`)
* **Multi-Distro Repository Management**: Added `tailor tools repo [add|rm]` (`cmd/tools_repo.go`, `pkg/repo/`) allowing users to easily configure or remove official `penguins-eggs.net` repositories and GPG signing keys on Debian/Ubuntu (DEB822 and classic list formats), Arch Linux/Manjaro (`pacman.conf` / `pacman-key`), Fedora/EL9 (`dnf`/`rpm`), openSUSE Leap (`zypper`), and Alpine Linux (`apk`).
* **Documentation & Reference Updates**: Documented repository management features and command usage examples in `README.md`.

### 🔄 Execution Flow & Sequence/Finalization Separation
* **Deferred Finalization**: Reordered the costume execution pipeline (`pkg/tailor/wear.go`) so that costume sysroot configuration sync and finalization scripts run strictly after all accessories have been installed and configured.
* **Sequence vs. Finalize Commands**: Added support for intermediate sequence commands (`sequence.cmds`) alongside finalization commands (`finalize.cmds`), while maintaining backward compatibility with legacy `cmds` syntax (`pkg/tailor/types.go`, `pkg/tailor/wear.go`).
* **Unit Test Coverage**: Added dedicated unit tests (`pkg/tailor/wear_test.go`) validating sequence and finalization command parsing and execution order.

### 🖥️ Interactive UX & Automated Reboot Handling
* **Interactive Terminal Review Pause**: Added an interactive prompt (`Press Enter to continue to summary report...`) at the conclusion of costume execution, allowing users to review logs and command outputs in the split-screen TUI before rendering the final summary box.
* **Automatic System Reboot**: When a costume specifies `reboot: true`, `tailor wear` now displays a reboot countdown notification and automatically triggers a system restart upon completion.

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
