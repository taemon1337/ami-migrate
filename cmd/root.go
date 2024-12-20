package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/taemon1337/ec-manager/pkg/config"
	"github.com/taemon1337/ec-manager/pkg/logger"
)

var (
	// Common flags
	instanceID string
	enabled    bool
	newAMI     string
	userID     string
	logLevel   string
	timeout    time.Duration
	defaultTimeout = 5 * time.Minute
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ecman",
	Short: "EC2 instance management tool",
	Long: `ec-manager (ecman) is a CLI tool that helps you manage your AWS EC2 instances.
It provides commands for:
- Creating new instances with proper configuration
- Listing and checking instance status
- Migrating instances to new AMIs
- Managing instance backups
- Cleaning up unused instances`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add flags that are used by multiple commands
	rootCmd.PersistentFlags().StringVar(&instanceID, "instance-id", "", "ID of the EC2 instance")
	rootCmd.PersistentFlags().BoolVar(&enabled, "enabled", false, "Only process instances with ami-migrate=enabled tag")
	rootCmd.PersistentFlags().StringVar(&newAMI, "new-ami", "", "ID of the new AMI to migrate to")
	rootCmd.PersistentFlags().StringVar(&userID, "user", "", "Your AWS username (defaults to current AWS user)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", defaultTimeout, "Timeout for AWS operations")

	// Initialize logger
	cobra.OnInitialize(initLogger)
}

// initLogger initializes the logger with the specified log level
func initLogger() {
	logger.Init(logger.LogLevel(logLevel))
}

// getUserID returns the user ID, either from flag or AWS credentials
func getUserID(cmd *cobra.Command) (string, error) {
	// Check if user flag is set
	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return "", err
	}

	// If user flag is set, use it
	if user != "" {
		return user, nil
	}

	// Try to get user from AWS credentials
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	awsUser, err := config.GetAWSUsername(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get AWS username: %v", err)
	}

	if awsUser == "" {
		return "", fmt.Errorf("--user flag is required when AWS username cannot be determined")
	}

	return awsUser, nil
}

// GetLogLevel returns the log level from flags
func GetLogLevel() string {
	return logLevel
}

// GetTimeout returns the timeout from flags
func GetTimeout() time.Duration {
	return timeout
}
