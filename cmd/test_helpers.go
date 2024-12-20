package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/taemon1337/ec-manager/pkg/client"
	"github.com/taemon1337/ec-manager/pkg/config"
	apitypes "github.com/taemon1337/ec-manager/pkg/types"
)

// setupTest creates a new mock client and command for testing
func setupTest(use string, setupMock func(*apitypes.MockEC2Client)) (*cobra.Command, *apitypes.MockEC2Client) {
	// Set timeout for tests
	config.SetTimeout(10 * time.Second)

	// Create mock client and set it
	mockClient := apitypes.NewMockEC2Client()
	if setupMock != nil {
		setupMock(mockClient)
	}
	client.SetEC2Client(mockClient)

	// Create a new command
	cmd := &cobra.Command{Use: use}

	// Add common flags
	cmd.Flags().String("instance-id", "", "ID of the instance")
	cmd.Flags().Bool("enabled", false, "Process all instances with ami-migrate=enabled tag")
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	cmd.Flags().Duration("timeout", 10*time.Second, "Timeout for AWS operations")

	return cmd, mockClient
}
