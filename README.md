# penguins-tailor

**penguins-tailor** is a standalone, lightweight tool written in Go to manage and apply system configurations, desktop environments, and themes ("costumes") to Linux distributions.

---

## 🚀 Features

- **Get**: Download or update the costumes repository (`tailor get`).
- **List**: Enumerate available costumes and their descriptions (`tailor list`).
- **Show**: Inspect detailed information and packages required by a costume (`tailor show <costume>`).
- **Wear**: Seamlessly apply a costume to the system (`sudo tailor wear <costume>`), configuring repositories, packages, sysroot overlays, and user skel settings.
- **Export**: Transfer native packages (`tailor export pkg`) or execution logs and reports (`tailor export log`) to remote storage via SSH.
- **Build**: Integrated packaging tool to compile binaries and produce native distribution packages (`tailor tools build`).
- **Distro-Aware**: Automatically identifies target distributions (Debian, Ubuntu, Arch, Alpine, Fedora, openSUSE, etc.) and generates assistance prompts if non-Debian package managers are present.

---

## 📦 Installation

```bash
git clone https://github.com/pieroproietti/penguins-tailor.git
cd penguins-tailor
make
sudo make install
```

---

## 👔 Command Reference

### Basic Commands

- **`tailor get [url]`**
  Clones or updates the costumes repository into `~/.wardrobe`. Defaults to `https://github.com/pieroproietti/penguins-wardrobe` if no URL is provided.
  ```bash
  # Default repository
  tailor get

  # Custom repository
  tailor get https://github.com/myuser/my-wardrobe
  ```

- **`tailor list`**
  Lists all available costumes found in the repository along with a brief description.
  ```bash
  tailor list
  ```

- **`tailor show <costume>`**
  Shows detailed metadata for a specific costume (e.g. description, supported distributions, packages, accessories, and commands).
  ```bash
  tailor show colibri
  ```

- **`tailor wear <costume>`**
  Applies the specified costume to the system. Requires root privileges (`sudo`).
  ```bash
  sudo tailor wear colibri
  ```
  **Flags:**
  - `--no-acc`: Skip installing accessory packages.
  - `--no-firm`: Skip installing firmware accessories.

---

### Export Commands

The **`export`** command suite automates the transfer of generated artifacts and logs to configured remote destinations:

#### 1. Export Native Packages (`tailor export pkg`)
Transfers compiled native packages (`.deb`, `.rpm`, `.pkg.tar.zst`, `.apk`) corresponding to the current distribution family to the remote storage server (`root@192.168.1.2:/eggs/`). It establishes an SSH multiplexed connection for efficient multi-file transfer.

```bash
# Export the built package
tailor export pkg

# Clean old versions on the remote server before exporting
tailor export pkg --clean
```

**Flags:**
- `--clean`: Removes previous versions of the package matching the distribution pattern on the remote server before uploading the new one.

#### 2. Export Logs and Reports (`tailor export log`)
Collects and uploads the main tailor log file (`/var/log/tailor.log`) and the latest detailed wear report (`/var/log/tailor/tailor-report-*.txt`) to the target server in a single SSH session without requiring manual file copying.

```bash
# Export logs to default remote destination
tailor export log

# Export logs with custom SSH user, IP, and destination directory
tailor export log -u artisan -i 192.168.1.50 -d /home/artisan/logs
```

**Flags:**
- `-u, --user <username>`: Remote SSH username (default: `artisan`).
- `-i, --ip <address>`: Remote IP address or hostname (default: `192.168.1.2`).
- `-d, --dir <path>`: Destination directory on the remote machine (default: `/home/artisan`).

---

### Packaging & Auxiliary Tools

- **`tailor tools build`**
  Compiles binaries and generates distribution-specific packages (`.deb` for Debian/Ubuntu, `PKGBUILD`/`.pkg.tar.zst` for Arch Linux, `.rpm` for Fedora/openSUSE, `.apk` for Alpine). Must be run as a regular user (not root).
  ```bash
  tailor tools build
  ```

---

## 📜 License

MIT License. Copyright (c) 2026 Piero Proietti.

