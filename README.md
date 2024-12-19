# AMI Migration Tool

A Go-based tool for managing AWS AMI migrations. This tool helps automate the process of updating EC2 instances to use a new AMI while maintaining proper versioning and backups.

## Features

- Automatic AMI version tracking using tags
- Parallel instance migration
- Volume snapshots for backup
- Graceful error handling and status tracking
- Clean separation of concerns with pkg structure
- Docker-based build and test environment
- Selective instance migration based on tags
- Migration of instances regardless of current AMI
- Smart instance state handling

## Prerequisites

- Docker
- Make

No need to install Go locally - all Go operations are performed within Docker containers!

## Quick Start

1. Build the project:
```bash
make build
```

2. Run tests:
```bash
make test
```

3. Build Docker image:
```bash
make docker-build
```

4. Run the tool:
```bash
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  ami-migrate:latest \
  --new-ami ami-xxxxx \
  --latest-tag latest \
  --enabled-value enabled
```

## CLI Arguments

- `--new-ami` (required): The ID of the new AMI to upgrade instances to
- `--latest-tag` (optional, default: "latest"): The tag used to track the current AMI version
- `--enabled-value` (optional, default: "enabled"): Value to match for the "ami-migrate" tag

## How It Works

1. **AMI Version Management**:
   - The tool tracks AMI versions using tags
   - When a new AMI is specified, the old "latest" AMI is archived
   - The new AMI becomes the "latest" version

2. **Instance Selection**:
   - Instances are selected for migration if they have `ami-migrate=enabled`
   - Running instances are migrated immediately
   - Stopped instances are handled based on additional tags

3. **Instance State Handling**:
   - Running instances: Migrated directly
   - Stopped instances with `ami-migrate-if-running=enabled`: 
     1. Instance is started
     2. Migration is performed
     3. Instance is returned to its original state
   - Stopped instances without the if-running tag: Skipped
   - Failed migrations are tagged for tracking

4. **Migration Process**:
   - Migrations run in parallel for faster updates
   - Each instance's volumes are snapshotted before migration
   - Instance state is preserved throughout the process
   - Comprehensive error handling and status tracking

## Instance Tagging

Two tags control the migration behavior:

1. Main Migration Tag:
```
Key: ami-migrate
Value: enabled  # or your custom value specified with --enabled-value
```

2. Optional State Control Tag:
```
Key: ami-migrate-if-running
Value: enabled
```

Tag Combinations and Behavior:
- Running instances:
  - Requires BOTH `ami-migrate=enabled` AND `ami-migrate-if-running=enabled`
  - Will be skipped if missing either tag
- Stopped instances:
  - Only requires `ami-migrate=enabled`
  - Will be migrated regardless of `ami-migrate-if-running` tag

This ensures that:
1. Running instances are only migrated when explicitly allowed via both tags
2. Stopped instances can be safely migrated with just the main migration tag

## Migration Status Tracking

The tool tracks migration status using the following tags:

1. Status Tag:
```
Key: ami-migrate-status
Value: [status]  # One of: skipped, in-progress, failed, warning, completed
```

2. Message Tag:
```
Key: ami-migrate-message
Value: [detailed message]  # Explains the current status
```

3. Timestamp Tag:
```
Key: ami-migrate-timestamp
Value: [UTC timestamp]  # Format: RFC3339
```

Status Values:
- `skipped`: Instance was not migrated (e.g., running instance without required tags)
- `in-progress`: Migration has started
- `failed`: Migration failed (error message in ami-migrate-message)
- `warning`: Migration partially successful (e.g., migrated but failed to stop)
- `completed`: Migration completed successfully

These tags provide a clear audit trail of the migration process and help identify any issues that need attention.

## Usage

The tool provides three main commands:

### 1. Migrate

Migrate instances to a new AMI version:

```bash
# Migrate instances by tag
ami-migrate migrate --new-ami ami-xxxxx

# Migrate specific instance
ami-migrate migrate --new-ami ami-xxxxx --instance-id i-xxxxx
```

Optional flags:
- `--latest-tag`: Tag value for the current latest AMI (default: "latest")
- `--enabled-value`: Value to match for the ami-migrate tag (default: "enabled")
- `--instance-id`: ID of specific instance to migrate (bypasses tag requirements)

### 2. Backup

Create snapshots of all volumes attached to instances:

```bash
# Backup instances by tag
ami-migrate backup

# Backup specific instance
ami-migrate backup --instance-id i-xxxxx
```

Optional flags:
- `--enabled-value`: Value to match for the ami-migrate tag (default: "enabled")
- `--instance-id`: ID of specific instance to backup (bypasses tag requirements)

The backup command will:
1. Find all instances with the ami-migrate tag (or use specified instance)
2. Create snapshots of all attached volumes
3. Tag snapshots with instance and device information

### 3. Restore

Restore a volume from a snapshot to an instance:

```bash
ami-migrate restore --snapshot-id snap-xxxxx --instance-id i-xxxxx
```

Required flags:
- `--snapshot-id`: ID of the snapshot to restore from
- `--instance-id`: ID of the instance to restore to

The restore command will:
1. Create a new volume from the snapshot
2. Stop the instance if it's running
3. Attach the volume to the instance using the original device name

### CI/CD Integration

For GitLab CI, add this to your `.gitlab-ci.yml`:

```yaml
ami-migrate:
  image: golang:1.21-alpine
  script:
    - go install github.com/taemon1337/ami-migrate@latest
    - ami-migrate migrate --new-ami $NEW_AMI_ID
  rules:
    - if: $CI_COMMIT_TAG  # Only run on tags
```

Make sure to set these environment variables in GitLab:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`
- `NEW_AMI_ID`

## Development

### Available Make Commands

- `make all` - Clean, build, and test
- `make build` - Build the binary
- `make clean` - Clean build artifacts
- `make test` - Run tests in Docker
- `make lint` - Run linter in Docker
- `make docker-build` - Build Docker image
- `make docker-test` - Run tests in Docker
- `make init` - Initialize go.mod (if needed)
- `make shell` - Open an interactive shell in the container

### Project Structure

```
ami-migrate/
├── Dockerfile          # Multi-stage Docker build
├── Makefile           # Build automation
├── main.go            # Entry point
├── pkg/
│   └── ami/           # AMI management package
│       ├── ami.go     # Core AMI operations
│       └── ami_test.go # Unit tests
```

### Adding New Features

1. Add new functionality to the appropriate package in `pkg/`
2. Write tests for new functionality
3. Update documentation as needed
4. Run tests using `make test`
5. Build and test the Docker image using `make docker-build`

## AWS Configuration

When running the containerized version, mount your AWS credentials:

```bash
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  ami-migrate:latest \
  --new-ami ami-xxxxx \
  --latest-tag latest \
  --enabled-value enabled
```

## License

MIT License