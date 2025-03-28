# darn - A Security Finding Remediation Tool

[![asciicast](https://asciinema.org/a/HD5ZFMWujd4cuDNAeeQ61COXp.svg)](https://asciinema.org/a/HD5ZFMWujd4cuDNAeeQ61COXp)

**NOTE:** This demo runs a little fast. I'll try to record it to make it a bit slower another time

Darn is a command-line tool designed to manage and enforce security best practices against projects. It provides a flexible framework for applying security remediation actions through templated files and CLI commands.

Darn and Darnit development is sponsored by Kusari, and is released under the Apache-2.0 License.

## Installation

### From Source

```bash
git clone https://github.com/kusari-oss/darn.git
cd darn
make all install
```

### Using Go

```bash
go install github.com/kusari-oss/darn/cmd/darn@latest
go install github.com/kusari-oss/darn/cmd/darnit@latest
```

## Quick Start

Initialize a new darn project:

```bash
# Initialize with default settings
darn init

# Initialize in a specific directory with custom settings
darn init my-darn-project --templates-dir=my-templates --actions-dir=my-actions
```

Run an action:

```bash
# Using a parameters file directly
darn action run add-security-md configs/params.json

# Using a JSON string after --
darn action run add-security-md -- {"repo":"my-repo","name":"My Project","emails":["security@example.com"]}
```

Using darnit:

Assuming there's a `report.json` with content:

```json
{
  "repository": "kusari-oss/darn-test-project",
  "scan_date": "2025-03-14T15:09:26Z",
  "scan_tool": "security-scanner-v1.0",
  "findings": {
    "security_policy": "missing",
    "code_of_conduct": "present",
    "mfa_status": "disabled",
    "branch_protection": "partial",
    "secrets_scanning": "enabled",
    "dependency_review": "disabled"
  },
  "risk_score": 65
}
```

and a `parameters.json` with content (these are test values for a fake project):

```json
{
  "project_name": "Darn Test Project",
  "organization": "kusari-oss",
  "repo_name": "darn-test-project",
  "security_contacts": ["security@example.com", "admin@example.com"]
}
```

```bash
darnit plan generate -m ~/.darn/library/mappings/security-remediation.yaml report.json --params params.json -o plan.json -v
```

This will generate a `plan.json` that should look like:

```json
{
  "project_name": "Darn Security Tool",
  "repository": "kusari-oss/darn",
  "steps": [
    {
      "id": "security-policy-remediation",
      "action_name": "create-branch",
      "params": {
        "branch_name": "add-security-docs"
      },
      "reason": "Create branch for security documentation"
    },
    {
      "id": "add-security-docs",
      "action_name": "add-security-md",
      "params": {
        "name": "Darn Security Tool",
        "emails": ["security@example.com", "admin@example.com"]
      },
      "reason": "Add SECURITY.md file",
      "depends_on": ["security-policy-remediation"]
    },
    {
      "id": "org-security-remediation",
      "action_name": "enable-mfa",
      "params": {
        "organization": "kusari-oss"
      },
      "reason": "Enable MFA requirement for the organization"
    }
    // Additional steps omitted for brevity
  ]
}
```

You can then execute the plan, assuming you have tools that are required, e.g. `git` and the github cli, `gh`:

```bash
darnit plan execute plan.json -v
```

## Core Concepts

- **Actions**: Reusable operations that implement security best practices
- **Templates**: Template files used by file actions
- **Parameters**: Values passed to actions (in JSON or YAML format)

## Command Reference

### Initialization

```bash
darn init [directory] [flags]
```

**Flags:**

- `--templates-dir`: Directory for template files (default: "templates")
- `--actions-dir`: Directory for action files (default: "actions")
- `--configs-dir`: Directory for configuration files (default: "configs")
- `--local-only`: Use only embedded defaults, don't attempt to fetch latest from remote
- `--remote-url`: URL for remote defaults repository

### Actions

```bash
# List available actions
darn action list

# Get detailed information about a specific action
darn action info [action-name]

# Run a specific action with parameters file
darn action run [action-name] [params-file]

# Run a specific action with JSON string
darn action run [action-name] -- [json-string]
```

### Darn Configuration

```bash
# Display current configuration
darn config show

# Create a new config file
darn config init
```

### Defaults

```bash
# Update defaults from remote source
darn defaults update

# List embedded default files
darn defaults list
```

## Action Types

Darn supports the following action types:

### File Actions

File actions create or modify files using templates:

```yaml
name: add-security-md
type: file
description: "Add SECURITY.md file to repository"
template_path: "security.md.tmpl"
target_path: "{{.repo}}/SECURITY.md"
create_dirs: true
schema: {
  "type": "object",
  "required": ["repo", "name", "emails"],
  "properties": {
    "repo": {
      "type": "string",
      "description": "Repository name"
    },
    "name": {
      "type": "string",
      "description": "Project name"
    },
    "emails": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Security contact emails"
    }
  }
}
```

### CLI Actions

CLI actions execute command-line tools (like GitHub CLI):

```yaml
name: enable-mfa
type: cli
description: "Enable MFA for the organization"
command: "gh"
args: 
  - "api"
  - "orgs/{{.organization}}"
  - "--jq"
  - ".two_factor_requirement_enabled"
schema: {
  "type": "object",
  "required": ["organization"],
  "properties": {
    "organization": {
      "type": "string",
      "description": "GitHub organization name"
    }
  }
}
```

### Git Actions

Darn includes CLI actions for Git operations:

```yaml
# Simple action to stage and commit changes
name: git-commit
type: cli
description: "Stage and commit changes to files"
command: "git"
args:
  - "add"
  - "{{.files}}"
  - "&&"
  - "git"
  - "commit"
  - "-m"
  - "{{.message}}"
schema: {
  "type": "object",
  "required": ["files", "message"],
  "properties": {
    "files": {
      "type": "string",
      "description": "Files to stage (space-separated list or glob pattern)"
    },
    "message": {
      "type": "string",
      "description": "Commit message"
    }
  }
}
```

Example usage:

```bash
# Using a parameters file
darn action run git-commit configs/commit-params.json

# Using inline JSON parameters
darn action run git-commit -- {"files": "README.md SECURITY.md", "message": "Add security documentation"}

# Stage and commit all changes with inline parameters
darn action run git-commit -- {"files": ".", "message": "Update all security settings"}
```

## Creating Custom Actions

1. Create a new YAML file in your actions directory
2. Define the action type, parameters schema, and other properties
3. For file actions, create the corresponding template file

## Configuration

Darn uses a YAML configuration file that can be specified with the `--config` flag:

```yaml
templates_dir: "templates"
actions_dir: "actions"
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License - see the LICENSE file for details.
