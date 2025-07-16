# Darn Library System Documentation

This document explains how the darn library system works, from basic concepts to technical implementation details.

## Overview

The darn library system provides a structured way to organize and manage reusable components for security remediation workflows. It consists of four main types of components:

- **Actions** - Executable operations (CLI commands, file creation)
- **Templates** - Reusable content patterns 
- **Configs** - Configuration presets
- **Mappings** - Rules that map security findings to remediation actions

## Library Structure

### Standard Directory Layout

```
library/
├── actions/          # Action definitions (.yaml files)
├── templates/        # Template files (.txt, .md, etc.)
├── configs/         # Configuration files (.yaml, .json)
└── mappings/        # Mapping rules (.yaml files)
```

### Library Path Resolution

The library path is resolved using this precedence order:

1. **Command line** - `--library-path` flag
2. **Environment** - `DARN_HOME` variable (for testing)
3. **Global config** - `~/.darn/config.yaml` setting
4. **Default** - `~/.darn/library`

Example:
```bash
# Use specific library
darn --library-path /custom/lib action list

# Use temporary library for testing
DARN_HOME=/tmp/test-lib darn action list

# Use global library (default)
darn action list
```

## Components Deep Dive

### Actions

Actions are the core executable units in darn. They define how to perform specific remediation tasks.

#### Action Types

**1. CLI Actions** (`type: "cli"`)
- Execute command-line tools
- Support parameter templating
- Validate commands exist in PATH

**2. File Actions** (`type: "file"`)  
- Create files from templates
- Support directory creation
- Safe path handling

#### Action Definition Structure

```yaml
# actions/example-action.yaml
name: "example-action"
description: "Example action that demonstrates the structure"
type: "cli"
command: "echo"
args: ["Hello {{.name}}!"]

parameters:
  - name: "name"
    type: "string"
    required: true
    description: "Name to greet"
    default: "World"

# Optional: outputs for use by other actions
outputs:
  greeting_file: "{{.working_dir}}/greeting.txt"

# Optional: labels for categorization
labels:
  category: ["demo", "example"]
  complexity: ["simple"]
```

#### Parameter Types and Validation

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text value | `"hello world"` |
| `array` | List of values | `["item1", "item2"]` |
| `number` | Numeric value | `42` |
| `boolean` | True/false | `true` |

Parameters support:
- **Required validation** - `required: true`
- **Default values** - `default: "value"`
- **Constraints** - Min/max for numbers, regex for strings
- **Templating** - Use `{{.param_name}}` in command args

### Templates

Templates are reusable content patterns used by file actions.

#### Template Structure

```
templates/
├── security.md           # Simple template
├── workflows/            # Organized by category
│   ├── github-actions.yml
│   └── ci-pipeline.yml
└── licenses/
    ├── apache-2.0.txt
    └── mit.txt
```

#### Template Syntax

Templates use Go's `text/template` syntax:

```markdown
<!-- templates/security.md -->
# Security Policy for {{.project_name}}

## Reporting Vulnerabilities

Please report security vulnerabilities to {{.security_email}}.

{{if .has_bug_bounty}}
## Bug Bounty Program

We offer rewards for qualifying security reports.
{{end}}

## Supported Versions

| Version | Supported |
|---------|-----------|
{{range .supported_versions}}
| {{.version}} | {{.status}} |
{{end}}
```

Used by file action:
```yaml
# actions/add-security-md.yaml
name: "add-security-md"
type: "file"
template_path: "security.md"
target_path: "SECURITY.md"
create_dirs: false

parameters:
  - name: "project_name"
    type: "string"
    required: true
  - name: "security_email"
    type: "string"
    required: true
  - name: "has_bug_bounty"
    type: "boolean"
    default: false
```

### Configs

Configuration files provide reusable settings and presets.

#### Example Config

```yaml
# configs/github-security.yaml
name: "GitHub Security Standards"
description: "Standard security configuration for GitHub repositories"

settings:
  branch_protection:
    enforce_admins: true
    required_status_checks:
      strict: true
      contexts: ["ci/tests", "security/scan"]
    required_pull_request_reviews:
      required_approving_review_count: 2
      dismiss_stale_reviews: true

  security_features:
    vulnerability_alerts: true
    security_updates: true
    secret_scanning: true
    
defaults:
  security_email: "security@company.com"
  response_time: "24 hours"
```

### Mappings

Mappings define rules that automatically select actions based on security findings.

#### Mapping Structure

```yaml
# mappings/security-baseline.yaml
mappings:
  - id: "missing-security-policy"
    condition: "security_policy == 'missing'"
    action: "add-security-md"
    reason: "Add required security policy documentation"
    parameters:
      project_name: "{{.project_name}}"
      security_email: "{{.security_email}}"

  - id: "weak-branch-protection" 
    condition: "branch_protection == 'none' || branch_protection == 'weak'"
    action: "enable-branch-protection"
    reason: "Strengthen branch protection rules"
    parameters:
      repository: "{{.repository}}"
      
  - id: "complex-remediation"
    condition: "security_score < 70"
    mapping_ref: "comprehensive-security.yaml"
    reason: "Apply comprehensive security improvements"
    parameters:
      baseline_config: "github-security"
```

#### Condition Syntax

Conditions use CEL (Common Expression Language):

```yaml
# Simple comparisons
condition: "mfa_enabled == false"
condition: "security_score < 50"

# Logical operators
condition: "mfa_enabled == false && admin_users > 0"
condition: "branch_protection == 'none' || branch_protection == 'weak'"

# String operations
condition: "security_policy.contains('incomplete')"
condition: "repository.startsWith('public-')"

# Array operations  
condition: "required_checks.size() < 2"
condition: "'security-scan' in required_checks"
```

#### Mapping References

Complex remediation can be split across multiple mapping files:

```yaml
# mappings/security-baseline.yaml
mappings:
  - id: "comprehensive-fix"
    condition: "needs_full_security_review == true"
    mapping_ref: "detailed-security-review.yaml"
    parameters:
      severity_level: "{{.severity}}"
```

## Library Management

### Initialization

Create a new library with standard structure:

```bash
# Initialize at default location (~/.darn/library)
darn library init

# Initialize at custom location
darn library init /path/to/custom/library

# Initialize with custom subdirectory names
darn library init --actions-dir my-actions --templates-dir my-templates
```

### Synchronization

Update library with latest embedded defaults:

```bash
# Sync global library
darn library sync

# Preview changes without applying
darn library sync --dry-run

# Sync specific library
darn library sync --library-path /custom/library

# Use only local defaults (no remote fetch)
darn library sync --local-only
```

### Management Commands

```bash
# Set global library path
darn library set-global /path/to/library

# Diagnose configuration issues
darn library diagnose

# Update from source directory
darn library update /source/directory

# List available actions
darn action list

# Show action details
darn action show create-file
```

## Technical Implementation

### Library Manager

The `library.Manager` handles path resolution and validation:

```go
type Manager struct {
    globalLibraryPath  string
    cmdLineLibraryPath string
    verboseLogging     bool
    validatedPaths     map[string]bool
}

// Resolve library path with clear precedence
func (m *Manager) ResolveLibraryPath() (*LibraryInfo, error)

// Validate library structure
func (m *Manager) validateLibraryPath(path, source string) *LibraryInfo
```

### Action Factory

The action factory creates action instances from definitions:

```go
type Factory struct {
    actionCreators map[string]ActionCreator
    context        ActionContext
}

// Register action types
func (f *Factory) RegisterDefaultTypes()

// Create action from config
func (f *Factory) Create(config Config) (Action, error)
```

### Execution Flow

1. **Library Resolution**
   - Resolve library path using precedence rules
   - Validate library structure exists
   - Load action definitions

2. **Action Creation**
   - Parse action YAML definition
   - Validate required parameters
   - Create action instance via factory

3. **Parameter Processing**
   - Merge defaults with provided values
   - Validate parameter types and constraints
   - Process template variables

4. **Execution**
   - CLI actions: Validate command exists, execute
   - File actions: Resolve template, create target file

### Error Handling

The system provides detailed error messages:

- **Library not found** - Clear resolution order and suggestions
- **Invalid action** - Specific validation failures
- **Missing parameters** - List of required parameters
- **Command not found** - PATH validation and installation hints

### Security Considerations

- **No shell injection** - CLI actions use exec.Command, not shell
- **Path validation** - Prevent directory traversal attacks
- **Input sanitization** - Template variables are escaped
- **Command validation** - Check commands exist before execution

## Best Practices

### Organizing Actions

```
actions/
├── security/
│   ├── add-security-md.yaml
│   ├── enable-mfa.yaml
│   └── setup-scanning.yaml
├── compliance/
│   ├── add-license.yaml
│   └── setup-governance.yaml
└── infrastructure/
    ├── setup-monitoring.yaml
    └── configure-alerts.yaml
```

### Template Organization

```
templates/
├── policies/
│   ├── security.md
│   ├── privacy.md
│   └── terms.md
├── workflows/
│   ├── ci.yml
│   └── security-scan.yml
└── configs/
    ├── eslint.json
    └── prettier.json
```

### Parameter Design

```yaml
parameters:
  # Use descriptive names
  - name: "organization_name"  # Good
    # not: "org"              # Too short
    
  # Provide defaults when sensible  
  - name: "response_time"
    default: "24 hours"
    
  # Use appropriate types
  - name: "admin_emails"
    type: "array"              # For multiple values
    # not: "string"            # Comma-separated string
    
  # Add descriptions
  - name: "severity_threshold"
    description: "Minimum severity level to trigger alerts (1-10)"
    type: "number"
```

### Mapping Strategy

```yaml
# Group related checks
mappings:
  # Basic security hygiene
  - id: "security-policy-check"
    condition: "security_policy == 'missing'"
    # ...
    
  - id: "license-check" 
    condition: "license == 'missing'"
    # ...
    
  # Advanced security (separate file)
  - id: "advanced-security"
    condition: "security_score < 80"
    mapping_ref: "advanced-security.yaml"
```

## Integration Examples

### CI/CD Pipeline

```yaml
# .github/workflows/security.yml
name: Security Remediation
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Run Security Scan
        run: security-scanner --output report.json
        
      - name: Generate Remediation Plan
        run: |
          darn library sync
          darnit plan generate report.json \
            --mapping security-baseline.yaml \
            --output remediation-plan.yaml
            
      - name: Execute Remediation
        run: darnit plan execute remediation-plan.yaml
```

### Custom Action Development

```bash
# 1. Create action definition
cat > ~/.darn/library/actions/custom-security-check.yaml << EOF
name: "custom-security-check"
description: "Run custom security validation"
type: "cli"
command: "my-security-tool"
args: ["--config", "{{.config_file}}", "--output", "{{.output_file}}"]

parameters:
  - name: "config_file"
    type: "string"
    required: true
  - name: "output_file" 
    type: "string"
    default: "security-report.json"
EOF

# 2. Test the action
darn action show custom-security-check

# 3. Use in mapping
cat > ~/.darn/library/mappings/custom-checks.yaml << EOF
mappings:
  - id: "run-custom-check"
    condition: "requires_custom_validation == true"
    action: "custom-security-check"
    parameters:
      config_file: "{{.security_config}}"
EOF
```

This documentation provides a complete understanding of how the darn library system works, from basic usage to advanced customization and integration patterns.