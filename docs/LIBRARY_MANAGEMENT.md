# Improved Library Management System

The darn library system has been enhanced with robust path resolution, validation, and error reporting to provide a more reliable and debuggable experience.

## Key Improvements

### 1. Clear Path Resolution Precedence

The library path is resolved using a clear hierarchy (highest to lowest priority):

1. **Command line flag** (`--library-path`)
2. **DARN_HOME environment variable** (for testing)
3. **Global config file setting** (`~/.darn/config.yaml`)
4. **Default global library** (`~/.darn/library`)

### 2. Robust Validation

- **Library structure validation**: Checks for required subdirectories (`actions/`, `templates/`, `configs/`, `mappings/`)
- **Path accessibility**: Verifies directories exist and are readable
- **CLI command validation**: Validates that CLI commands exist in PATH (for `cli` action type)
- **Cross-platform support**: Handles Windows executable extensions

### 3. Better Error Reporting

- **Detailed error messages**: Clear indication of what went wrong and where
- **Verbose logging**: Optional detailed output for debugging
- **Diagnostic command**: Built-in troubleshooting tool

## Usage

### Diagnosing Issues

When experiencing library or shell command issues, use the diagnostic command:

```bash
# Basic diagnostics
darn library diagnose

# Verbose output for debugging
darn library diagnose -v

# JSON output for automation
darn library diagnose --json
```

### Library Switching for Testing

For testing or project-specific libraries, use the `DARN_HOME` environment variable:

```bash
# Use a temporary library for testing
DARN_HOME=/tmp/test-library darn library diagnose

# Set up a project-specific library
export DARN_HOME=/path/to/project-library
darn action list
```

### Setting Up Libraries

```bash
# Initialize a new library at default location
darn library init

# Initialize at custom location
darn library init /path/to/custom/library

# Set global library path
darn library set-global /path/to/custom/library

# Sync library with latest embedded defaults
darn library sync

# Sync with verbose output
darn library sync --verbose

# Preview what would be synced (dry run)
darn library sync --dry-run
```

## Technical Details

### Library Manager

The new `library.Manager` provides:

- **Path resolution** with environment-specific handling
- **Validation** of library structure and shell commands  
- **Diagnostics** for troubleshooting
- **Cross-platform** compatibility

### Configuration Integration

The enhanced `Config` struct includes:

- `LibraryManager` for robust path handling
- `ValidateLibrarySetup()` for upfront validation
- `GetLibraryDiagnostics()` for debugging information
- `ValidateShellCommand()` for command validation

### Error Handling

Instead of silent failures, the system now:

- **Validates** library paths at startup
- **Reports** specific error conditions
- **Provides** actionable recommendations
- **Logs** resolution attempts in verbose mode

## Troubleshooting

### Common Issues

1. **Library not found**
   ```
   Solution: Run `darn library init` or `darn library set-global <path>`
   ```

2. **CLI commands not found**
   ```
   Solution: Install missing commands or check PATH environment variable
   ```

3. **Permission errors**
   ```
   Solution: Check directory permissions and user access rights
   ```

### Environment-Specific Problems

The diagnostic command identifies:

- Path resolution issues
- Missing directories
- Permission problems
- Shell command availability
- Environment variable settings

### Testing Different Configurations

Use `DARN_HOME` for temporary testing:

```bash
# Create test library
mkdir -p /tmp/test-lib/{actions,templates,configs,mappings}

# Test with temporary library
DARN_HOME=/tmp/test-lib darn library diagnose

# Run operations with test library
DARN_HOME=/tmp/test-lib darn action list
```

## CLI-Based Library Management

The new CLI-based approach provides several commands for managing libraries:

- **`darn library init`** - Initialize new libraries with standard structure
- **`darn library sync`** - Update library with latest embedded defaults
- **`darn library diagnose`** - Troubleshoot library configuration issues
- **`darn library set-global`** - Configure global library path
- **`darn library update`** - Update library from source directory

### Benefits over shell scripts:
- ✅ **Cross-platform** - Works on Windows, macOS, Linux
- ✅ **Integrated validation** - Checks paths and permissions
- ✅ **Consistent interface** - Same CLI patterns as other commands
- ✅ **Better error handling** - Clear, actionable error messages
- ✅ **Dry-run support** - Preview changes before applying

## Migration from Old System

The improved system is backward compatible, but provides:

- **Better error messages** when things go wrong
- **Validation** that catches issues early
- **Diagnostics** for troubleshooting problems
- **Consistent behavior** across environments
- **CLI-based management** instead of shell scripts

Existing configurations will continue to work, but you'll get better feedback when there are issues.

