# Darn Library Quick Start Guide

This guide helps you get started with the darn library system quickly.

## 5-Minute Setup

### 1. Initialize Your Library

```bash
# Create library at default location (~/.darn/library)
darn library init

# Verify it's working
darn library diagnose
```

### 2. List Available Actions

```bash
# See all available actions
darn action list

# Look for specific actions
darn action list | grep security
```

### 3. Use an Action

```bash
# Show action details
darn action show add-security-md

# Run an action (this will be implemented in action execution)
# darn action run add-security-md --project-name "My Project" --security-email "security@example.com"
```

## Common Tasks

### Adding Custom Actions

```bash
# Create a simple CLI action
cat > ~/.darn/library/actions/hello-world.yaml << 'EOF'
name: "hello-world"
description: "Simple greeting action"
type: "cli" 
command: "echo"
args: ["Hello {{.name}}!"]

parameters:
  - name: "name"
    type: "string"
    required: true
    description: "Name to greet"
EOF

# Test it
darn action show hello-world
```

### Creating File Actions

```bash
# Create template
cat > ~/.darn/library/templates/readme.md << 'EOF'
# {{.project_name}}

{{.description}}

## Installation

```bash
npm install {{.package_name}}
```

## Usage

See examples in the documentation.

## Contributing

Please read CONTRIBUTING.md for contribution guidelines.
EOF

# Create action that uses the template
cat > ~/.darn/library/actions/add-readme.yaml << 'EOF'
name: "add-readme"
description: "Add README.md file to project"
type: "file"
template_path: "readme.md"
target_path: "README.md"
create_dirs: false

parameters:
  - name: "project_name"
    type: "string"
    required: true
  - name: "description"
    type: "string"
    required: true
  - name: "package_name"
    type: "string"
    required: false
    default: "{{.project_name}}"
EOF
```

### Library Management

```bash
# Update library with latest defaults
darn library sync

# Preview what would be updated
darn library sync --dry-run

# Set custom library location
darn library set-global /path/to/custom/library

# Troubleshoot issues
darn library diagnose --verbose
```

### Working with Different Libraries

```bash
# Use temporary library for testing
mkdir -p /tmp/test-lib/{actions,templates,configs,mappings}

# Test with temporary library
DARN_HOME=/tmp/test-lib darn action list

# Copy action to test library
cp ~/.darn/library/actions/add-security-md.yaml /tmp/test-lib/actions/

# Use test library
DARN_HOME=/tmp/test-lib darn action show add-security-md
```

## Action Examples

### 1. Simple CLI Action

```yaml
# actions/git-init.yaml
name: "git-init"
description: "Initialize git repository"
type: "cli"
command: "git"
args: ["init", "{{.directory}}"]

parameters:
  - name: "directory"
    type: "string"
    required: false
    default: "."
```

### 2. File Creation Action

```yaml
# actions/create-gitignore.yaml
name: "create-gitignore"
description: "Create .gitignore file"
type: "file"
template_path: "gitignore.txt"
target_path: ".gitignore"
create_dirs: false

parameters:
  - name: "language"
    type: "string"
    required: true
    description: "Programming language (node, python, go, etc.)"
```

### 3. Action with Outputs

```yaml
# actions/create-config.yaml
name: "create-config"
description: "Create configuration file"
type: "file"
template_path: "config.json"
target_path: "{{.config_dir}}/{{.config_name}}.json"
create_dirs: true

parameters:
  - name: "config_name"
    type: "string"
    required: true
  - name: "config_dir"
    type: "string"
    default: "config"
  - name: "environment"
    type: "string"
    default: "development"

outputs:
  config_path: "{{.config_dir}}/{{.config_name}}.json"
```

## Template Examples

### 1. Simple Template

```markdown
<!-- templates/license-mit.txt -->
MIT License

Copyright (c) {{.year}} {{.author}}

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction...
```

### 2. Conditional Template

```yaml
# templates/github-workflow.yml
name: {{.workflow_name}}

on:
  push:
    branches: [ {{.main_branch}} ]
  pull_request:
    branches: [ {{.main_branch}} ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      {{if .run_tests}}
      - name: Run Tests
        run: {{.test_command}}
      {{end}}
      
      {{if .run_security_scan}}
      - name: Security Scan
        run: {{.security_command}}
      {{end}}
```

### 3. Loop Template

```markdown
<!-- templates/team-contacts.md -->
# Team Contacts

{{range .team_members}}
## {{.name}}
- Role: {{.role}}
- Email: {{.email}}
{{if .slack_handle}}- Slack: {{.slack_handle}}{{end}}

{{end}}
```

## Mapping Examples

### 1. Basic Mapping

```yaml
# mappings/basic-security.yaml
mappings:
  - id: "add-security-policy"
    condition: "security_policy == 'missing'"
    action: "add-security-md"
    reason: "Repository needs security policy"
    parameters:
      project_name: "{{.project_name}}"
      security_email: "security@company.com"

  - id: "add-license"
    condition: "license == 'missing'"
    action: "add-license-mit"
    reason: "Repository needs license"
    parameters:
      author: "{{.author}}"
      year: "{{.current_year}}"
```

### 2. Complex Conditions

```yaml
# mappings/advanced-security.yaml
mappings:
  - id: "enable-mfa"
    condition: "mfa_enabled == false && admin_count > 0"
    action: "setup-mfa"
    reason: "MFA required for repositories with admin users"

  - id: "branch-protection"
    condition: "branch_protection == 'none' || (branch_protection == 'basic' && sensitivity == 'high')"
    action: "enable-branch-protection"
    reason: "Strengthen branch protection for sensitive repository"
```

## Troubleshooting

### Common Issues

**Library not found:**
```bash
darn library diagnose
# Check the path resolution and create library if needed
darn library init
```

**Action not found:**
```bash
darn action list | grep action-name
# If missing, check if it exists in library
ls ~/.darn/library/actions/
```

**Command not found (CLI actions):**
```bash
darn library diagnose
# Check "Command Availability" section
# Install missing commands or update PATH
```

**Template not found (File actions):**
```bash
ls ~/.darn/library/templates/
# Ensure template file exists and path is correct in action definition
```

### Debug Mode

```bash
# Verbose library operations
darn library sync --verbose

# Detailed diagnostics
darn library diagnose --verbose --json
```

### Testing Changes

```bash
# Always test with dry-run first
darn library sync --dry-run

# Use test library for experiments
DARN_HOME=/tmp/test-lib darn action show my-test-action
```

## Next Steps

1. **Read the full documentation**: See `LIBRARY_SYSTEM.md` for complete details
2. **Create custom actions**: Start with simple CLI or file actions
3. **Set up mappings**: Define rules for automatic remediation
4. **Integrate with workflows**: Use in CI/CD pipelines or scripts

## Getting Help

- **Diagnose issues**: `darn library diagnose`
- **Check configuration**: `darn library diagnose --json`
- **List actions**: `darn action list`
- **Action details**: `darn action show <action-name>`