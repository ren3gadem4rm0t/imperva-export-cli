# Imperva Export CLI

[![Build Status](https://github.com/ren3gadem4rm0t/imperva-export-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/ren3gadem4rm0t/imperva-export-cli/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/ren3gadem4rm0t/imperva-export-cli/branch/main/graph/badge.svg?token=YOUR_CODECOV_TOKEN)](https://codecov.io/gh/ren3gadem4rm0t/imperva-export-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/ren3gadem4rm0t/imperva-export-cli)](https://goreportcard.com/report/github.com/ren3gadem4rm0t/imperva-export-cli)
![License](https://img.shields.io/github/license/ren3gadem4rm0t/imperva-export-cli)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/ren3gadem4rm0t/imperva-export-cli?sort=semver)
![GitHub issues](https://img.shields.io/github/issues/ren3gadem4rm0t/imperva-export-cli)
![GitHub stars](https://img.shields.io/github/stars/ren3gadem4rm0t/imperva-export-cli?style=social)
![Gosec](https://img.shields.io/badge/gosec-passing-brightgreen)

The **Imperva Export CLI** is a command-line tool designed to interact seamlessly with the Imperva Account-Export API. It facilitates the export of account configurations into ZIP files formatted for standard Terraform usage. This tool simplifies the process of exporting, monitoring, and downloading account configurations, ensuring efficient and automated management of your Imperva resources.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
  - [Commands](#commands)
    - [Export](#export)
    - [Download](#download)
    - [Status](#status)
    - [Auto](#auto)
- [Logging](#logging)
- [Error Handling](#error-handling)
- [Development](#development)
  - [Testing](#testing)
  - [Linting](#linting)
  - [Code Coverage](#code-coverage)
  - [Static Analysis](#static-analysis)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

## Features

- **Export Account Configurations**: Initiate export processes for parent or sub-accounts.
- **Asynchronous Operations**: Handle long-running export tasks efficiently.
- **Status Monitoring**: Poll and monitor the status of export processes.
- **Secure Downloads**: Safely download exported ZIP files with validation.
- **Flexible Configuration**: Configure via environment variables, configuration files, or command-line flags.
- **Structured Logging**: Utilize structured logging with adjustable verbosity levels.
- **Graceful Shutdown**: Supports interrupt signals to safely terminate operations.
- **Automated Testing**: Comprehensive test coverage ensures reliability.

## Installation

You can install the Imperva Export CLI using `go install`:

```bash
go install github.com/ren3gadem4rm0t/imperva-export-cli@latest
```

### Prerequisites

- **Go**: Ensure you have Go installed (version 1.23 or later). You can download it from [Go's official website](https://golang.org/dl/).

### Clone the Repository

```bash
git clone https://github.com/ren3gadem4rm0t/imperva-export-cli.git
cd imperva-export-cli
```

### Build the Binary

```bash
go build -o imperva-export-cli ./main.go
```

This command compiles the CLI tool and produces an executable named `imperva-export-cli` in the current directory.

### Install via Makefile

Alternatively, you can use the provided `Makefile` for building and managing the project.

```bash
make build
```

## Configuration

The CLI tool requires authentication credentials to interact with the Imperva API. These can be provided through environment variables, a configuration file, or command-line flags.

### Authentication Credentials

- **API ID**: A unique identifier for your API access.
- **API Key**: A secret key associated with your API ID.

### Methods to Provide Credentials

1. **Environment Variables**

   Set the following environment variables:

   ```bash
   export API_ID=your-api-id
   export API_KEY=your-api-key
   ```

2. **Configuration File**

   Create a configuration file named `imperva-export-cli.yaml` in the `$HOME/.config/` directory with the following content:

   ```yaml
   api-id: your-api-id
   api-key: your-api-key
   log-level: info
   output-dir: /path/to/output
   ```

3. **Command-Line Flags**

   Provide credentials directly via command-line flags when executing commands:

   ```bash
   imperva-export-cli export --caid 123456 --api-id your-api-id --api-key your-api-key
   ```

### Configuration Hierarchy

The CLI prioritizes configuration sources in the following order:

1. **Command-Line Flags**
2. **Environment Variables**
3. **Configuration File**
4. **Defaults**

### Additional Configuration Options

- **Log Level**: Adjust the verbosity of logs.
  - Flags: `--log-level`
  - Environment Variable: `LOG_LEVEL`
  - Options: `none`, `debug`, `info`, `warn`, `error`

- **Output Directory**: Specify where exported files are saved.
  - Flags: `--output-dir`
  - Environment Variable: `OUTPUT_DIR`
  - Default: Current directory (`.`)

## Usage

The CLI provides several commands to manage the export process. Below are detailed descriptions and examples for each command.

### Commands

#### Export

**Description**: Initiates the export process for a specified CAID (Customer Account ID).

**Usage**:

```bash
imperva-export-cli export --caid <CAID> [flags]
```

**Flags**:

- `--caid`: *(Required)* The account ID to export configurations for.
- `--api-id`: API ID (optional if set via environment/config).
- `--api-key`: API Key (optional if set via environment/config).
- `--log-level`: Set log verbosity (`none`, `debug`, `info`, `warn`, `error`).
- `--output-dir`: Directory to save exported files.

**Example**:

```bash
imperva-export-cli export --caid 123456
```

#### Download

**Description**: Downloads the exported ZIP file using the provided handler and CAID.

**Usage**:

```bash
imperva-export-cli download --caid <CAID> --handler <HANDLER> [flags]
```

**Flags**:

- `--caid`: *(Required)* The account ID associated with the export.
- `--handler`: *(Required)* The handler ID received during export initiation.
- `--api-id`: API ID (optional if set via environment/config).
- `--api-key`: API Key (optional if set via environment/config).
- `--log-level`: Set log verbosity (`none`, `debug`, `info`, `warn`, `error`).
- `--output-dir`: Directory to save the downloaded file.

**Example**:

```bash
imperva-export-cli download --caid 123456 --handler abc-def-ghi-jkl
```

#### Status

**Description**: Checks the status of an ongoing export process using the handler and CAID.

**Usage**:

```bash
imperva-export-cli status --caid <CAID> --handler <HANDLER> [flags]
```

**Flags**:

- `--caid`: *(Required)* The account ID associated with the export.
- `--handler`: *(Required)* The handler ID received during export initiation.
- `--api-id`: API ID (optional if set via environment/config).
- `--api-key`: API Key (optional if set via environment/config).
- `--log-level`: Set log verbosity (`none`, `debug`, `info`, `warn`, `error`).

**Example**:

```bash
imperva-export-cli status --caid 123456 --handler abc-def-ghi-jkl
```

#### Auto

**Description**: Automates the entire export process by initiating the export, monitoring its status, and downloading the exported file upon completion.

**Usage**:

```bash
imperva-export-cli auto --caid <CAID> [flags]
```

**Flags**:

- `--caid`: *(Required)* The account ID to export configurations for.
- `--api-id`: API ID (optional if set via environment/config).
- `--api-key`: API Key (optional if set via environment/config).
- `--log-level`: Set log verbosity (`none`, `debug`, `info`, `warn`, `error`).
- `--output-dir`: Directory to save exported files.

**Example**:

```bash
imperva-export-cli auto --caid 123456
```

### Common Flags Across Commands

- `--api-id`: Provide API ID directly.
- `--api-key`: Provide API Key directly.
- `--log-level`: Control log verbosity.
- `--output-dir`: Specify where to save exported files.

## Logging

The Imperva Export CLI uses [zerolog](https://github.com/rs/zerolog) for structured logging. You can control the verbosity of logs using the `--log-level` flag or the `LOG_LEVEL` environment variable.

**Log Levels**:

- `none`: Disable all logs.
- `debug`: Detailed debug information.
- `info`: General operational messages.
- `warn`: Warning messages indicating potential issues.
- `error`: Error messages indicating failed operations.

**Examples**:

- **Set to Debug Level**:

  ```bash
  imperva-export-cli export --caid 123456 --log-level debug
  ```

- **Set via Environment Variable**:

  ```bash
  export LOG_LEVEL=debug
  imperva-export-cli export --caid 123456
  ```

**Default Log Level**: `none`

## Error Handling

The CLI tool provides comprehensive error messages to help diagnose issues during operations. Errors can occur due to:

- **Invalid Inputs**: Incorrect CAID or handler formats.
- **Authentication Failures**: Missing or invalid API credentials.
- **API Errors**: Issues returned by the Imperva API, such as rate limiting or server errors.
- **Network Issues**: Connectivity problems during API requests.
- **File System Errors**: Issues with writing or saving exported files.

**Best Practices**:

- Ensure that the CAID and handler provided are correct and correspond to existing accounts/export processes.
- Verify that API credentials (`API_ID` and `API_KEY`) are valid and have the necessary permissions.
- Check network connectivity if experiencing persistent errors during API requests.
- Ensure that the specified output directory is writable and has sufficient space.

**Example Error Message**:

```bash
Error: failed to initiate export: API error: Authentication Error - Authentication missing or invalid (Status Code: 401)
```

## Development

### Prerequisites

- **Go**: Version 1.23 or later.
- **Make**: For using the provided `Makefile`.
- **Static Analysis Tools**: Optional but recommended (e.g., `golangci-lint`, `gosec`).

### Building the Project

Use the `Makefile` for streamlined building and managing the project.

- **Build the Project**:

  ```bash
  make build
  ```

- **Run Tests**:

  ```bash
  make test
  ```

- **Run Linting**:

  ```bash
  make lint
  ```

- **Format Code**:

  ```bash
  make fmt
  ```

- **Check Code Formatting**:

  ```bash
  make check-fmt
  ```

- **Run Static Analysis**:

  ```bash
  make staticcheck
  make ast
  ```

- **Generate Coverage Report**:

  ```bash
  make coverage
  ```

- **Clean Build Artifacts**:

  ```bash
  make clean
  ```

### Testing

The project includes comprehensive unit tests covering various components and scenarios.

- **Run All Tests**:

  ```bash
  make test
  ```

- **View Test Coverage**:

  ```bash
  make coverage
  ```

Coverage reports are generated in the `coverage/` directory and can be viewed in HTML format.

### Linting

Ensure code quality and adherence to best practices using `golangci-lint`.

- **Run Linting**:

  ```bash
  make lint
  ```

### Code Coverage

Monitor test coverage to ensure critical paths are tested.

- **Generate Coverage Report**:

  ```bash
  make coverage
  ```

- **View Coverage in Browser**:

  The coverage report is saved as `coverage/coverage.html` and can be opened in a web browser.

### Static Analysis

Perform security and static code analysis using tools like `gosec` and `staticcheck`.

- **Run GoSec**:

  ```bash
  make ast
  ```

- **Run StaticCheck**:

  ```bash
  make staticcheck
  ```

### Documentation

Generate and view documentation using `godoc`.

- **Start Godoc Server**:

  ```bash
  make docs
  ```

  Access the documentation at [http://localhost:6060/pkg/github.com/ren3gadem4rm0t/imperva-export-cli/internal/cmd/](http://localhost:6060/pkg/github.com/ren3gadem4rm0t/imperva-export-cli/internal/cmd/)

## Contributing

Contributions are welcome! Please follow the guidelines below to contribute to the project.

### Guidelines

1. **Fork the Repository**: Create a personal fork of the repository.
2. **Clone Your Fork**:

   ```bash
   git clone https://github.com/your-username/imperva-export-cli.git
   cd imperva-export-cli
   ```

3. **Create a New Branch**:

   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Changes**: Implement your feature or fix.
5. **Run Tests and Linting**:

   ```bash
   make test
   make lint
   ```

6. **Commit Changes**:

   ```bash
   git commit -m "Add feature: your feature description"
   ```

7. **Push to Your Fork**:

   ```bash
   git push origin feature/your-feature-name
   ```

8. **Create a Pull Request**: Submit a pull request through GitHub.

## License

This project is licensed under the [MIT License](LICENSE).

## Support

If you encounter any issues or have questions, please open an issue on the [GitHub repository](https://github.com/ren3gadem4rm0t/imperva-export-cli/issues).
