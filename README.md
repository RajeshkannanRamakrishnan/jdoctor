# jdoctor 🩺

**jdoctor** is a CLI tool designed to help Java developers quickly identify configuration problems, version mismatches, build health issues, and SSL header analysis in their local environment.

It acts as a comprehensive health check for your Java development setup, ensuring that your tools (Java, Maven, Gradle) are correctly configured and working in harmony.

## Features

- **🔍 Full System Diagnosis**: Checks Java version, OS details, build tool health (Maven/Gradle), and dependency conflicts.
- **🔐 SSL/TLS Analysis**: detailed analysis of SSL certificates for any host, including validation against local Java truststores to detect corporate proxy/MITM issues.
- **☕ Java Process List (`ps`)**: Enhanced process listing specifically for Java applications, showing uptime and summarized arguments.
- **🐚 Project REPL (`repl`)**: Launches `jshell` with your project's classpath pre-loaded (Maven/Gradle), allowing you to experiment with your project's code interactively.
- **📦 Dependency Scan (`deps`)**: Scans `pom.xml` for potential duplicate or conflicting dependencies.

## Installation

### From Source

Requirements: Go 1.22+

```bash
git clone https://github.com/RajeshkannanRamakrishnan/jdoctor.git
cd jdoctor
go install ./cmd/jdoctor
```

## Usage

### 1. Full Health Check
Run a complete scan of your environment:
```bash
jdoctor doctor
```
**Output includes:**
- Detected Java version & Architecture.
- Maven/Gradle version and health.
- Dependency conflict summary.

### 2. SSL Diagnosis
Diagnose connection issues to a remote host. Useful for debugging "PKIX path building failed" errors.

**Basic Check:**
```bash
jdoctor ssl google.com
```

**Detailed Diagnosis:**
```bash
jdoctor ssl diagnose google.com
```
**Checks performed:**
- Certificate chain validation.
- Trust validation against OS (Go) system roots.
- **Trust validation against local Java truststore**.
- Expiration checks and MITM detection (e.g., Zscaler, BlueCoat).

### 3. Smart Process List
List running Java processes with readable output:
```bash
jdoctor ps
```

### 4. Interactive REPL
Start a JShell session with your current project's dependencies loaded:
```bash
cd /path/to/my-java-project
jdoctor repl
```

### 5. Dependency Analysis
Check for conflicts in your `pom.xml`:
```bash
jdoctor deps
```

## Contributing
Contribution is welcome!
