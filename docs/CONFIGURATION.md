# Darn Configuration Management

This document details how Darn and Darnit manage configurations, focusing on the library path used for templates, actions, and other library components. Darn now operates with a **global-only configuration model**, centered around `~/.darn/config.yaml`.

## Configuration Hierarchy

Darn/Darnit determine which library to use based on the following hierarchy (highest priority first):

1.  **`darnit --library-path <path>` Flag (for `darnit` tool):**
    *   When running `darnit`, this command-line flag overrides all other library configurations. Darnit will exclusively use the library specified by `<path>`.
    *   Purpose: Useful for temporarily testing a specific library or when you need to ensure a particular library version is used for a single `darnit` execution, without altering global settings.
    *   Example: `darnit plan --report findings.json --library-path /path/to/custom-darn-library`

2.  **Global Configuration (`~/.darn/config.yaml`):**
    *   If the `darnit --library-path` flag is not used, Darn/Darnit consults the global configuration file located at `~/.darn/config.yaml`.
    *   The `library_path` field within this file dictates the active global library.
    *   This global configuration is managed using the `darn library set-global <path>` command.
    *   Purpose: Allows users to define a default shared library for all Darn operations.
    *   Example (`~/.darn/config.yaml` after running `darn library set-global /path/to/my-global-darn-library`):
        ```yaml
        library_path: /path/to/my-global-darn-library # This path is expanded, e.g., ~ is resolved
        use_global: true
        use_local: false 
        # Other settings like templates_dir, actions_dir might also be present
        templates_dir: templates
        actions_dir: actions
        ```

3.  **Default Built-in Library Path (`~/.darn/library`):**
    *   If the `darnit --library-path` flag is not used, and the global configuration file (`~/.darn/config.yaml`) is missing or does not contain a valid `library_path`, Darn/Darnit falls back to using the default library path: `~/.darn/library`.
    *   Purpose: Provides a built-in default location for a shared library if no other configuration is present. The `darn library init` command (with no arguments) and `darn library update` (with no arguments and no active global config) will target this path.

## Library Management Commands

### `darn library init [target-library-path]`

The `darn library init` command initializes a darn library structure at a specified location. It **only creates and populates the library content** (actions, templates, etc.) and **does not modify any configuration files** or set the initialized library as the active one.

*   **Behavior:**
    *   If `[target-library-path]` is provided (e.g., `~/my-custom-lib`, `./a-local-lib`), the library structure is created and populated at that specified path. The path can be absolute or relative. Tilde `~` is expanded to the user's home directory.
    *   If no `[target-library-path]` is provided, it defaults to initializing the library content at `~/.darn/library`.
    *   The command creates standard subdirectories (e.g., `actions/`, `templates/`, `configs/`, `mappings/`) within the target library path and populates them with default content from embedded resources or a remote URL (unless `--local-only` is specified).
*   **Important:** This command **does not** create or modify `~/.darn/config.yaml`. To make the initialized library the active global library, you must use `darn library set-global <target-library-path>` subsequently.
*   **Flags:**
    *   `--templates-dir`, `--actions-dir`, etc.: Customize subdirectory names within the library.
    *   `--local-only`: Use only embedded defaults, do not fetch from remote.
    *   `--remote-url`: Specify a custom URL for remote defaults.
    *   `--verbose`: Enable verbose output.
*   **Examples:**
    1.  Initialize the default global library content:
        ```bash
        darn library init
        # Library content created at ~/.darn/library
        # To make this your active global library:
        darn library set-global ~/.darn/library
        ```
    2.  Initialize library content in a custom directory:
        ```bash
        darn library init /opt/shared/my-darn-lib
        # Library content created at /opt/shared/my-darn-lib
        # To make this your active global library:
        darn library set-global /opt/shared/my-darn-lib
        ```
    3.  Initialize library content in a directory relative to the current location:
        ```bash
        darn library init ./darn-assets
        # Library content created at ./darn-assets
        # To make this your active global library (path will be resolved to absolute):
        darn library set-global ./darn-assets
        ```

### `darn library set-global <path>`

The `darn library set-global <path>` command sets or updates the active global library path in the user's global configuration file (`~/.darn/config.yaml`).

*   **Action:**
    *   Writes to or creates the global configuration file at `~/.darn/config.yaml`.
    *   Sets the `library_path` field to the provided `<path>`. The path is expanded (e.g., `~` resolves to the home directory) and stored as an absolute path.
    *   Sets `use_global: true` and `use_local: false` in the global config to ensure this library is used by default.
    *   If the `~/.darn` directory doesn't exist, it will be created.
*   **Impact:** After running this command, Darn/Darnit operations (outside of `darnit` calls using `--library-path`) will use the library at this configured path.
*   **Example:**
    ```bash
    darn library set-global ~/my-main-darn-library
    # ~/.darn/config.yaml is updated:
    # library_path: /home/user/my-main-darn-library (actual expanded path)
    # use_global: true
    # use_local: false
    ```

### `darn library update [source-directory]`

The `darn library update` command updates an existing darn library with new or modified content from a source directory.

*   **Behavior:**
    *   **With `--library-path <target-lib>`:** If this flag is provided, the library at `<target-lib>` is updated using content from `[source-directory]` (defaults to current directory if `[source-directory]` is omitted).
        ```bash
        darn library update --library-path /path/to/my-library ./my-source-updates
        ```
    *   **Without `--library-path`:**
        1.  It first checks the global configuration (`~/.darn/config.yaml`) for an active `library_path`. If found, this library is updated.
        2.  If the global config is missing, or `library_path` is not set, it defaults to updating the library content at `~/.darn/library`.
        The `[source-directory]` (defaults to current directory) is used as the source of updates.
        ```bash
        darn library update ./my-source-updates 
        # (Updates active global lib, or ~/.darn/library if none set, from ./my-source-updates)
        ```
*   **Flags:**
    *   `--force`: Force update even if files are identical.
    *   `--dry-run`: Show what would be updated without making changes.
    *   `--verbose`: Enable verbose output.

## Usage Scenarios

1.  **Setting up and using a global Darn library:**
    *   First, initialize the library content (e.g., in a custom location):
        ```bash
        darn library init ~/darn-global-lib
        ```
    *   Then, set this library as your active global library:
        ```bash
        darn library set-global ~/darn-global-lib
        ```
    *   Now, `darn` and `darnit` commands will use `~/darn-global-lib` by default.
        ```bash
        darn action list 
        darnit plan --report findings.json 
        ```

2.  **Updating your active global library from local changes:**
    *   Assume your active global library is `~/darn-global-lib`.
    *   You have some new templates or actions in `./my-library-additions`.
        ```bash
        darn library update ./my-library-additions
        # This updates ~/darn-global-lib with content from ./my-library-additions
        ```

3.  **Updating a specific, non-active library:**
    ```bash
    darn library update --library-path /some/other/darn-library ./source-for-other-lib
    ```

4.  **Temporarily using a different library for a `darnit` task:**
    ```bash
    # Your global library is ~/darn-global-lib, but for this one task:
    darnit plan --report security-report.json --library-path /temporary/test-library
    ```

This global-only configuration model simplifies how Darn manages its library, providing a clear and consistent approach.
