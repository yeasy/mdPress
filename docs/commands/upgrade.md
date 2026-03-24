# `mdpress upgrade`

[中文说明](upgrade_zh.md)

## Purpose

Check for newer versions of mdpress from GitHub releases and optionally install them automatically. The `upgrade` command simplifies keeping your mdpress installation up-to-date by automating version checking and binary replacement.

## Syntax

```bash
mdpress upgrade [flags]
```

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--check` | off | Only check for updates without installing. Useful for CI/CD pipelines or regular health checks. |
| `-v, --verbose` | off | Print detailed logs during the upgrade process. |
| `-q, --quiet` | off | Print errors only. |

## Behavior

### Version Check and Download

The `upgrade` command performs these steps:

1. **Fetch Latest Release**: Queries the GitHub API for the latest mdpress release
2. **Compare Versions**: Compares the latest version with your current version using semantic versioning
3. **Platform Detection**: Automatically detects your OS and architecture (Linux, macOS, Windows; x86_64, arm64)
4. **Download Binary**: Downloads the appropriate pre-built binary for your platform
5. **Backup Current**: Creates a backup of your current mdpress binary (with `.backup` extension)
6. **Replace Binary**: Installs the new binary with executable permissions
7. **Cleanup**: Removes the backup on successful installation

If an error occurs during installation, the command automatically attempts to restore from the backup.

### Version Detection

The upgrade command:

- Uses semantic versioning (e.g., `1.2.3`) for comparison
- Strips the `v` prefix from GitHub tags if present
- Compares version components numerically
- Recognizes when you're already running the latest version

### Supported Platforms

The upgrade command automatically supports:

- **Linux**: x86_64 (amd64), ARM64 (aarch64)
- **macOS**: x86_64 (Intel), ARM64 (Apple Silicon)
- **Windows**: x86_64 (amd64), ARM64 (aarch64)

If no binary is found for your platform in the latest release, the upgrade will fail with a clear error message.

## Examples

```bash
# Check for updates without installing
mdpress upgrade --check

# Install latest version (default behavior)
mdpress upgrade

# Install with verbose output for troubleshooting
mdpress upgrade --verbose

# Verify upgrade completed
mdpress --version
```

## Environment Variables

The `upgrade` command respects standard HTTP environment variables:

| Variable | Description |
| --- | --- |
| `HTTP_PROXY` | HTTP proxy URL for release downloads |
| `HTTPS_PROXY` | HTTPS proxy URL for release downloads |
| `NO_PROXY` | Comma-separated domains to bypass proxy |

## Exit Codes

| Code | Meaning |
| --- | --- |
| 0 | Success (either updated or already latest version) |
| 1 | Network error or failed to fetch release information |
| 2 | Version comparison or parsing error |
| 3 | Download or installation failed |

## Notes

- **Backup Creation**: Before replacing the binary, a backup is created with the `.backup` extension. On successful installation, this backup is automatically removed.
- **Automatic Restoration**: If installation fails, the command automatically restores from the backup.
- **Manual Restoration**: If restoration fails, the backup location is shown so you can restore it manually.
- **Binary Location**: The upgrade replaces the binary at the location returned by `os.Executable()`, which is the path of the running mdpress binary.
- **Permissions**: The new binary is written with executable permissions (`0755`).
- **GitHub Rate Limiting**: The GitHub API allows 60 requests per hour per IP for unauthenticated requests. Upgrades should not hit this limit.
- **Network Connectivity**: The command requires internet access to check for and download new releases.
- **Installation Privileges**: On Linux/macOS, you may need elevated privileges to replace the binary if it's installed in system directories (e.g., `/usr/local/bin/`).

## Common Issues and Solutions

### "No binary found for your platform"

This error occurs when the latest GitHub release doesn't have a precompiled binary for your OS/architecture combination.

**Solution:**
- Check if your platform is [supported](#supported-platforms)
- Build from source: `go install github.com/yeasy/mdpress@latest`
- Wait for the next release that includes your platform

### "Cannot determine current executable path"

This error occurs when the system cannot determine where the mdpress binary is located.

**Solution:**
- Run mdpress directly (not through a symlink or wrapper script)
- Run `which mdpress` to verify the binary location
- Try using the full path: `/usr/local/bin/mdpress upgrade`

### "Failed to create backup" or "Failed to install binary"

Installation failed, possibly due to file permissions.

**Solution:**
- Check file permissions: `ls -l $(which mdpress)`
- Try with elevated privileges: `sudo mdpress upgrade`
- Check disk space: `df -h`
- Restore from backup manually: `mv mdpress.backup mdpress`

### "Failed to fetch latest release" with proxy error

Network error, possibly due to proxy or firewall.

**Solution:**
- Configure HTTP proxy if needed:
  ```bash
  export HTTPS_PROXY=https://proxy.example.com:8080
  mdpress upgrade
  ```
- Try with verbose output: `mdpress upgrade --verbose`
- Test network connectivity: `curl -I https://api.github.com`

### Backup file left behind after failed upgrade

A `.backup` file exists but the new binary didn't install properly.

**Solution:**
```bash
# Check the current state
ls -la $(which mdpress)*

# If the new binary works, remove the backup
rm $(which mdpress).backup

# If the new binary is broken, restore from backup
mv $(which mdpress).backup $(which mdpress)
chmod +x $(which mdpress)
mdpress --version
```

## See Also

- [build](build.md) - Build outputs from Markdown
- [doctor](doctor.md) - Check system readiness
- [COMMANDS.md](../COMMANDS.md) - Command overview
