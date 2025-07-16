# Darn Architecture Overview

This document provides a high-level overview of how darn components work together.

## System Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Security      │    │     Report      │    │   Parameters    │
│   Scanner       │───▶│   findings.json │◀───│   params.json   │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │     darnit      │
                       │ plan generate   │
                       └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Mappings      │◀── Library System
                       │   Rules Engine  │    ┌─────────────────┐
                       └─────────────────┘    │     Actions     │
                                │             │   Templates     │
                                ▼             │    Configs      │
                       ┌─────────────────┐    │   Mappings      │
                       │ Remediation     │    └─────────────────┘
                       │ Plan (JSON)     │
                       └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │     darnit      │
                       │ plan execute    │
                       └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Remediation   │
                       │   Actions       │
                       │   Executed      │
                       └─────────────────┘
```

## Component Breakdown

### 1. Input Layer

**Security Scanner**
- External tools (e.g., GitHub Security Scanner, Snyk, etc.)
- Generates structured findings in JSON format
- Examples: Missing security policies, vulnerable dependencies, weak configurations

**Parameters**
- Project-specific variables (project name, maintainer email, etc.)
- Environment settings (production, staging, development)
- Policy preferences (security levels, compliance requirements)

### 2. Processing Layer

**darnit plan generate**
- Takes security findings + parameters as input
- Uses mapping rules to determine appropriate remediation actions
- Generates executable remediation plan
- Handles dependency resolution between actions

**Mapping Rules Engine**
- CEL (Common Expression Language) for condition evaluation
- Maps security findings to specific remediation actions
- Supports complex conditions and nested mappings
- Extensible rule definitions

### 3. Library System

**Actions**
- CLI commands (execute external tools)
- File operations (create/modify files from templates)
- Parameterized and reusable

**Templates** 
- Reusable content patterns
- Go template syntax with variables
- Organized by category (security policies, licenses, etc.)

**Mappings**
- Condition → Action mappings
- Hierarchical rule definitions
- Reference other mappings for complex scenarios

**Configs**
- Reusable configuration presets
- Environment-specific settings
- Policy templates

### 4. Execution Layer

**darnit plan execute**
- Executes actions defined in remediation plan
- Handles action dependencies and ordering
- Provides execution feedback and error handling
- Supports dry-run mode for testing

### 5. Management Layer

**darn CLI**
- Library management (init, sync, diagnose)
- Action inspection and testing
- Configuration management
- Cross-platform support

## Data Flow

### 1. Detection Phase
```
Scanner → findings.json + params.json
```

### 2. Planning Phase
```
findings.json + params.json + mappings → remediation-plan.json
```

### 3. Execution Phase
```
remediation-plan.json + library → executed actions
```

## Key Design Principles

### 1. **Separation of Concerns**
- **Detection**: External scanners focus on finding issues
- **Planning**: darnit focuses on determining what to do
- **Execution**: Actions focus on how to fix issues
- **Management**: darn CLI focuses on library organization

### 2. **Extensibility**
- Plugin architecture for actions
- Template-based content generation
- Rule-based mapping system
- Library-based organization

### 3. **Safety**
- Dry-run capabilities for testing
- Input validation and sanitization
- No shell injection vulnerabilities
- Explicit dependency management

### 4. **Flexibility**
- Multiple library support (global, project-specific, testing)
- Environment variable overrides
- Configurable precedence rules
- Cross-platform compatibility

## Security Model

### Input Validation
- JSON schema validation for findings and parameters
- Template variable sanitization
- Path validation for file operations
- Command existence verification

### Execution Safety
- No shell command injection
- Controlled command execution via exec.Command
- File system boundary enforcement
- Permission validation

### Library Isolation
- Separate library instances for different contexts
- Environment-based overrides for testing
- Validation of library structure and contents
- Secure defaults for new libraries

## Extension Points

### Custom Actions
```yaml
# Custom action definition
name: "custom-security-check"
type: "cli"
command: "my-tool"
args: ["--flag", "{{.value}}"]
parameters:
  - name: "value"
    type: "string"
    required: true
```

### Custom Templates
```go
// Template with custom logic
{{if .environment == "production"}}
  production-specific content
{{else}}
  development content
{{end}}
```

### Custom Mappings
```yaml
# Custom mapping rule
mappings:
  - id: "custom-check"
    condition: "custom_field == 'trigger_value'"
    action: "custom-action"
    parameters:
      value: "{{.custom_parameter}}"
```

### Integration APIs
```go
// Programmatic usage
config := darnit.GenerateOptions{
    MappingsDir: "/custom/mappings",
    ExtraParams: map[string]any{
        "environment": "production",
    },
}

plan, err := darnit.GenerateRemediationPlan(report, mappingFile, config)
```

## Scalability Considerations

### Performance
- Parallel action execution where possible
- Efficient CEL expression evaluation
- Template compilation and caching
- Library path resolution caching

### Large-Scale Deployment
- Centralized library management
- Policy as code through mappings
- Batch processing capabilities
- Integration with CI/CD systems

### Monitoring
- Execution logging and metrics
- Action success/failure tracking
- Performance monitoring
- Library usage analytics

This architecture enables darn to provide a flexible, secure, and scalable solution for automated security remediation while maintaining clear separation of concerns and extensibility.