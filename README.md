# AMI Migration Tool

A Go-based tool for managing AWS AMI migrations. This tool helps automate the process of updating EC2 instances to use a new AMI while maintaining proper versioning and backups.

## Quick Start

1. Install the tool:
```bash
docker pull ami-migrate:latest
```

2. Run your first migration:
```bash
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  ami-migrate:latest \
  migrate \
  --new-ami ami-xxxxx
```

## Core Features

- Automatic AMI version tracking and management
- Safe instance migration with volume snapshots
- Selective migration based on instance state and tags
- Comprehensive status tracking and error handling
- Automatic user detection from AWS credentials

## Common Tasks

### 1. List Your Instances
```bash
# Uses your AWS credentials username
ami-migrate list

# Or specify a different username
ami-migrate list --user johndoe
```

Output shows:
- Instance name and ID
- OS type and instance size
- Current state (running/stopped)
- IP addresses
- Current and latest AMI versions
- Migration status

### 2. Check Migration Status
```bash
# Uses your AWS credentials username
ami-migrate check

# Or specify a different username
ami-migrate check --user johndoe
```

Shows for each instance:
- Current AMI details
- Latest available AMI
- Migration recommendation

### 3. Create New Instance
```bash
# Create default Ubuntu instance
ami-migrate create

# Create custom RHEL instance
ami-migrate create \
  --os RHEL9 \
  --size xlarge \
  --name my-instance
```

Options:
- `--os`: Ubuntu (default) or RHEL9
- `--size`: small, medium, large, xlarge
- `--name`: Custom instance name
- `--user`: Optional, defaults to AWS credentials username

### 4. Migrate Instances
```bash
# Migrate by tag
ami-migrate migrate --new-ami ami-xxxxx

# Migrate specific instance
ami-migrate migrate \
  --new-ami ami-xxxxx \
  --instance-id i-xxxxx
```

The migration process:
1. Takes volume snapshots for backup
2. Stops the instance if running
3. Creates new instance with target AMI
4. Copies all tags
5. Terminates old instance
6. Starts new instance if original was running

### 5. Delete Instances
```bash
# Delete using AWS credentials username
ami-migrate delete --instance i-1234567890abcdef0

# Or specify a different username
ami-migrate delete \
  --user johndoe \
  --instance i-1234567890abcdef0
```

Safely deletes with:
1. Ownership verification
2. Instance details display
3. Confirmation prompt
4. Status updates

## Instance Tags

Control migration behavior with these tags:

1. Main Migration Tag (Required):
```
Key: ami-migrate
Value: enabled
```

2. Running Instance Control (Optional):
```
Key: ami-migrate-if-running
Value: enabled
```

3. Owner Tag (Required for check/list):
```
Key: Owner
Value: <your-aws-username>
```

Tag Requirements:
- Running instances need BOTH `ami-migrate=enabled` AND `ami-migrate-if-running=enabled`
- Stopped instances only need `ami-migrate=enabled`
- Owner tag is automatically set to your AWS username when creating instances

## Migration Status Tracking

Status is tracked via tags:

1. Status Tag:
```
Key: ami-migrate-status
Value: skipped | in-progress | failed | warning | completed
```

2. Message Tag:
```
Key: ami-migrate-message
Value: [detailed status message]
```

## Developer Information

### Prerequisites
- Docker
- Make
- AWS credentials configured (for automatic user detection)

### Build and Test
```bash
# Build project
make build

# Run tests
make test

# Build Docker image
make docker-build
```

### Project Structure
```
ami-migrate/
├── cmd/               # CLI commands
│   ├── backup.go     
│   ├── check.go      
│   ├── create.go     
│   ├── delete.go     
│   ├── list.go       
│   ├── migrate.go    
│   └── root.go       
├── pkg/
│   ├── ami/          # Core functionality
│   │   ├── ami.go    
│   │   ├── ami_test.go
│   │   └── mock_ec2.go
│   └── config/       # Configuration
│       └── aws.go    # AWS credentials handling
```

### Adding Features
1. Add service functions in `pkg/ami/ami.go`
2. Add unit tests in `pkg/ami/ami_test.go`
3. Update mock client if needed
4. Create CLI command in `cmd/`
5. Update documentation

## Usage Notes

All commands support automatic user detection from your AWS credentials. The `--user` flag is optional and only needed if you want to operate on instances owned by a different user.

The tool will attempt to get your username in the following order:
1. From the `--user` flag if provided
2. From your AWS credentials file
3. From the IAM user info
4. From the STS caller identity

This means you can run most commands without explicitly specifying your username:

```bash
# List your instances
ami-migrate list

# Check migration status
ami-migrate check

# Create a new instance
ami-migrate create --os RHEL9

# Delete an instance
ami-migrate delete --instance i-xxxxx
```

## CI/CD Integration

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

## AWS Configuration

When running the containerized version, mount your AWS credentials:

```bash
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  ami-migrate:latest \
  migrate \
  --new-ami ami-xxxxx
```

## License

MIT License