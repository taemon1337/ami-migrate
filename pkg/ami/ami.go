package ami

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2ClientAPI defines the interface for EC2 client operations
type EC2ClientAPI interface {
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	CreateSnapshot(ctx context.Context, params *ec2.CreateSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.CreateSnapshotOutput, error)
	TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	StopInstances(ctx context.Context, params *ec2.StopInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
	StartInstances(ctx context.Context, params *ec2.StartInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
}

// Service provides AMI management operations
type Service struct {
	client EC2ClientAPI
}

// NewService creates a new AMI service
func NewService(client EC2ClientAPI) *Service {
	return &Service{
		client: client,
	}
}

// GetAMIWithTag gets an AMI by its tag
func (s *Service) GetAMIWithTag(ctx context.Context, tagKey, tagValue string) (string, error) {
	input := &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + tagKey),
				Values: []string{tagValue},
			},
		},
	}

	result, err := s.client.DescribeImages(ctx, input)
	if err != nil {
		return "", fmt.Errorf("describe images: %w", err)
	}

	if len(result.Images) == 0 {
		return "", nil
	}

	return aws.ToString(result.Images[0].ImageId), nil
}

// TagAMI tags an AMI with the specified key and value
func (s *Service) TagAMI(ctx context.Context, amiID, tagKey, tagValue string) error {
	input := &ec2.CreateTagsInput{
		Resources: []string{amiID},
		Tags: []types.Tag{
			{
				Key:   aws.String(tagKey),
				Value: aws.String(tagValue),
			},
		},
	}

	_, err := s.client.CreateTags(ctx, input)
	return err
}

// MigrateInstances migrates instances to new AMI if they have the enabled tag
func (s *Service) MigrateInstances(ctx context.Context, oldAMI, newAMI, enabledValue string) error {
	instances, err := s.fetchEnabledInstances(ctx, enabledValue)
	if err != nil {
		return fmt.Errorf("fetch instances: %w", err)
	}

	if len(instances) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	for _, instance := range instances {
		shouldMigrate, needsStart := s.shouldMigrateInstance(instance)
		if !shouldMigrate {
			s.tagInstanceStatus(ctx, instance, "skipped", "Instance state or tags do not meet migration criteria")
			continue
		}

		wg.Add(1)
		go func(inst types.Instance, start bool) {
			defer wg.Done()

			s.tagInstanceStatus(ctx, inst, "in-progress", "Starting migration")

			// If instance needs to be started
			if needsStart && inst.State.Name != types.InstanceStateNameRunning {
				if err := s.startInstance(ctx, inst); err != nil {
					s.tagInstanceStatus(ctx, inst, "failed", fmt.Sprintf("Failed to start instance: %v", err))
					return
				}
			}

			// Perform migration
			if err := s.upgradeInstance(ctx, newAMI, inst); err != nil {
				s.tagInstanceStatus(ctx, inst, "failed", fmt.Sprintf("Failed to upgrade instance: %v", err))
				return
			}

			// If we started the instance, stop it again
			if needsStart && inst.State.Name != types.InstanceStateNameRunning {
				if err := s.stopInstance(ctx, inst); err != nil {
					s.tagInstanceStatus(ctx, inst, "warning", fmt.Sprintf("Migration successful but failed to stop instance: %v", err))
					return
				}
			}

			s.tagInstanceStatus(ctx, inst, "completed", "Migration completed successfully")
		}(instance, needsStart)
	}
	wg.Wait()

	return nil
}

func (s *Service) fetchEnabledInstances(ctx context.Context, enabledValue string) ([]types.Instance, error) {
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:ami-migrate"),
				Values: []string{enabledValue},
			},
		},
	}

	resp, err := s.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, err
	}

	var instances []types.Instance
	for _, reservation := range resp.Reservations {
		instances = append(instances, reservation.Instances...)
	}
	return instances, nil
}

func (s *Service) shouldMigrateInstance(instance types.Instance) (bool, bool) {
	isRunning := instance.State.Name == types.InstanceStateNameRunning
	hasIfRunningTag := false

	// Check for if-running tag
	for _, tag := range instance.Tags {
		if aws.ToString(tag.Key) == "ami-migrate-if-running" &&
			aws.ToString(tag.Value) == "enabled" {
			hasIfRunningTag = true
			break
		}
	}

	// If instance is running, we need both tags
	if isRunning {
		return hasIfRunningTag, false
	}

	// If instance is stopped, we only need ami-migrate tag (which is already checked in fetchEnabledInstances)
	return true, false
}

func (s *Service) startInstance(ctx context.Context, instance types.Instance) error {
	input := &ec2.StartInstancesInput{
		InstanceIds: []string{aws.ToString(instance.InstanceId)},
	}
	_, err := s.client.StartInstances(ctx, input)
	if err != nil {
		return err
	}

	// Wait for instance to start
	waiter := ec2.NewInstanceRunningWaiter(s.client)
	return waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{aws.ToString(instance.InstanceId)},
	}, 5*time.Minute)
}

func (s *Service) stopInstance(ctx context.Context, instance types.Instance) error {
	input := &ec2.StopInstancesInput{
		InstanceIds: []string{aws.ToString(instance.InstanceId)},
	}
	_, err := s.client.StopInstances(ctx, input)
	if err != nil {
		return err
	}

	// Wait for instance to stop
	waiter := ec2.NewInstanceStoppedWaiter(s.client)
	return waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{aws.ToString(instance.InstanceId)},
	}, 5*time.Minute)
}

func (s *Service) upgradeInstance(ctx context.Context, newAMI string, instance types.Instance) error {
	// Create snapshot of the instance's volumes
	for _, mapping := range instance.BlockDeviceMappings {
		if mapping.Ebs != nil {
			_, err := s.client.CreateSnapshot(ctx, &ec2.CreateSnapshotInput{
				VolumeId: mapping.Ebs.VolumeId,
				Description: aws.String(fmt.Sprintf("Backup before AMI migration for instance %s",
					aws.ToString(instance.InstanceId))),
			})
			if err != nil {
				return fmt.Errorf("create snapshot: %w", err)
			}
		}
	}

	// Stop the instance
	if instance.State.Name == types.InstanceStateNameRunning {
		if err := s.stopInstance(ctx, instance); err != nil {
			return fmt.Errorf("stop instance: %w", err)
		}
	}

	// Create new instance with new AMI
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(newAMI),
		InstanceType: instance.InstanceType,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	runResult, err := s.client.RunInstances(ctx, runInput)
	if err != nil {
		return fmt.Errorf("run instances: %w", err)
	}

	// Terminate old instance
	_, err = s.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{aws.ToString(instance.InstanceId)},
	})
	if err != nil {
		return fmt.Errorf("terminate instance: %w", err)
	}

	// Copy tags to new instance
	if err := s.copyTags(ctx, instance, runResult.Instances[0]); err != nil {
		return fmt.Errorf("copy tags: %w", err)
	}

	return nil
}

func (s *Service) copyTags(ctx context.Context, oldInstance, newInstance types.Instance) error {
	var tags []types.Tag
	for _, tag := range oldInstance.Tags {
		// Skip the migration status tag
		if aws.ToString(tag.Key) == "ami-migrate-status" {
			continue
		}
		tags = append(tags, tag)
	}

	input := &ec2.CreateTagsInput{
		Resources: []string{aws.ToString(newInstance.InstanceId)},
		Tags:      tags,
	}

	_, err := s.client.CreateTags(ctx, input)
	return err
}

func (s *Service) tagInstanceStatus(ctx context.Context, instance types.Instance, status, message string) error {
	input := &ec2.CreateTagsInput{
		Resources: []string{aws.ToString(instance.InstanceId)},
		Tags: []types.Tag{
			{
				Key:   aws.String("ami-migrate-status"),
				Value: aws.String(status),
			},
			{
				Key:   aws.String("ami-migrate-message"),
				Value: aws.String(message),
			},
			{
				Key:   aws.String("ami-migrate-timestamp"),
				Value: aws.String(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	}

	_, err := s.client.CreateTags(ctx, input)
	return err
}
