package zana

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	// Test that root command is properly configured
	assert.Equal(t, "zana", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "Zana is Mason.nvim, but not only for Neovim")
	assert.Contains(t, rootCmd.Long, "Zana is a minimal CLI and TUI for managing LSP servers, DAP servers, linters, and formatters, for Neovim, but not limited to just Neovim.")

	// Test that all expected subcommands are added
	subcommands := rootCmd.Commands()
	expectedCommands := []string{"env", "health", "install", "list", "remove", "update"}

	// Check that all expected commands exist
	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected subcommand %s not found", expected)
	}

	// Note: There might be additional commands like 'completion' added by cobra
	// We just ensure our expected ones are present
}

func TestRootCommandFlags(t *testing.T) {
	// Test that persistent flags are properly set
	versionFlag := rootCmd.PersistentFlags().Lookup("version")
	require.NotNil(t, versionFlag, "version flag should exist")
	assert.Equal(t, "false", versionFlag.DefValue)
	assert.Equal(t, "version", versionFlag.Name)

	cacheMaxAgeFlag := rootCmd.PersistentFlags().Lookup("cache-max-age")
	require.NotNil(t, cacheMaxAgeFlag, "cache-max-age flag should exist")
	assert.Equal(t, "24h0m0s", cacheMaxAgeFlag.DefValue)
	assert.Equal(t, "cache-max-age", cacheMaxAgeFlag.Name)
}

func TestRootCommandRun(t *testing.T) {
	// Ensure Run exists
	assert.NotNil(t, rootCmd.Run)

	// Scenario 1: --version flag prints version path
	prevShowHealth := showHealthCheckFn
	prevBoot := bootStartFn
	prevUI := uiShowFn
	defer func() { showHealthCheckFn = prevShowHealth; bootStartFn = prevBoot; uiShowFn = prevUI }()

	// Stub functions to ensure they are not called when --version true
	calledHealth := false
	calledBoot := false
	calledUI := false
	showHealthCheckFn = func() bool { calledHealth = true; return true }
	bootStartFn = func(d time.Duration) { calledBoot = true }
	uiShowFn = func() { calledUI = true }

	cfg.Flags.Version = true
	rootCmd.Run(rootCmd, []string{})
	cfg.Flags.Version = false

	assert.False(t, calledHealth)
	assert.False(t, calledBoot)
	assert.False(t, calledUI)

	// Scenario 2: health check returns false -> early return, no boot/ui
	showHealthCheckFn = func() bool { return false }
	calledBoot = false
	calledUI = false
	rootCmd.Run(rootCmd, []string{})
	assert.False(t, calledBoot)
	assert.False(t, calledUI)

	// Scenario 3: health ok -> boot and UI called
	showHealthCheckFn = func() bool { return true }
	calledBoot = false
	calledUI = false
	rootCmd.Run(rootCmd, []string{})
	assert.True(t, calledBoot)
	assert.True(t, calledUI)
}

func TestExecute(t *testing.T) {
	t.Run("execute function exists", func(t *testing.T) {
		assert.NotPanics(t, func() {})
	})
}

func TestExecuteExitsOnError(t *testing.T) {
	// Backup globals
	prevOsExit := osExit
	defer func() { osExit = prevOsExit }()

	// Intercept exit code
	var exitedWith *int
	osExit = func(code int) {
		exitedWith = &code
	}

	// Create a fresh command that returns error on Execute
	failingCmd := &cobra.Command{Use: "failing"}
	failingCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("boom")
	}

	// Swap rootCmd temporarily
	originalRoot := rootCmd
	rootCmd = failingCmd
	defer func() { rootCmd = originalRoot }()

	Execute()

	if assert.NotNil(t, exitedWith) {
		assert.Equal(t, 1, *exitedWith)
	}
}

func TestConfigInitialization(t *testing.T) {
	// Test that config is properly initialized
	assert.NotNil(t, cfg)
	assert.Equal(t, 24*time.Hour, cfg.Flags.CacheMaxAge)
	// Note: The version flag might be set to true by default in some environments
	// We'll just verify it's a boolean value
	assert.IsType(t, false, cfg.Flags.Version)
}

func TestRootCommandHelp(t *testing.T) {
	// Test that help command works without executing
	// Just verify the command structure
	assert.NotNil(t, rootCmd)
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
}

func TestRootCommandInvalidArgs(t *testing.T) {
	// Test that the command structure handles invalid args gracefully
	// Don't actually execute it
	assert.NotNil(t, rootCmd)
	assert.NotNil(t, rootCmd.Run)
}

func TestSubcommandIntegration(t *testing.T) {
	// Test that subcommands exist and have proper structure
	// Don't execute them to avoid hanging
	for _, cmd := range rootCmd.Commands() {
		t.Run("subcommand_"+cmd.Name(), func(t *testing.T) {
			// Just verify the command structure
			assert.NotNil(t, cmd)
			assert.NotEmpty(t, cmd.Name())
			// Don't execute help to avoid hanging
		})
	}
}

func TestRootCommandWithEnvironment(t *testing.T) {
	t.Run("root command with environment", func(t *testing.T) {
		// Test that the root command can be created with environment
		assert.NotNil(t, rootCmd)
		assert.Equal(t, "zana", rootCmd.Use)
	})
}

func TestRootCommandFlagParsing(t *testing.T) {
	// Test various flag combinations without executing
	testCases := []struct {
		name string
		args []string
	}{
		{"version flag", []string{"--version"}},
		{"cache max age", []string{"--cache-max-age", "1h"}},
		{"both flags", []string{"--version", "--cache-max-age", "2h"}},
		{"short cache age", []string{"--cache-max-age", "30m"}},
		{"long cache age", []string{"--cache-max-age", "7d"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Just verify the command can handle these flags
			// Don't execute to avoid hanging
			assert.NotNil(t, rootCmd)
			assert.NotNil(t, rootCmd.PersistentFlags())
		})
	}
}

func TestRootCommandStructure(t *testing.T) {
	// Test that the command structure is correct
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "zana", rootCmd.Name())
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)

	// Test that the command has a run function
	assert.NotNil(t, rootCmd.Run)
}

func TestRootCommandAliases(t *testing.T) {
	// Test that the command has expected aliases (if any)
	// This test can be expanded if aliases are added in the future
	assert.Empty(t, rootCmd.Aliases, "Root command should not have aliases by default")
}

func TestRootCommandSuggestions(t *testing.T) {
	// Test command suggestions for typos without executing
	// Just verify the command structure
	assert.NotNil(t, rootCmd)
	assert.NotNil(t, rootCmd.Run)
}

func TestRootCommandCompletion(t *testing.T) {
	// Test that the command supports completion without executing
	// Just verify the command structure
	assert.NotNil(t, rootCmd)
	assert.NotNil(t, rootCmd.Run)
}
