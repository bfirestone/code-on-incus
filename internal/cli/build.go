package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/image"
	"github.com/spf13/cobra"
)

var (
	buildForce bool
)

var buildCmd = &cobra.Command{
	Use:   "build [sandbox|privileged|custom]",
	Short: "Build Incus images for Claude sessions",
	Long: `Build opinionated Incus images for running Claude Code.

Available images:
  sandbox     - Sandbox image (Docker + build tools + Claude CLI)
  privileged  - Privileged image (sandbox + GitHub CLI + SSH)
  custom      - Custom image from user script

Examples:
  coi build sandbox
  coi build privileged
  coi build sandbox --force
  coi build custom my-image --script setup.sh
`,
	Args: cobra.MinimumNArgs(1),
	RunE: buildCommand,
}

// buildCustomCmd builds a custom image from a script
var buildCustomCmd = &cobra.Command{
	Use:   "custom <name>",
	Short: "Build a custom image from a user script",
	Long: `Build a custom image from a base image using a user-provided build script.

The build script should be a bash script that will be executed as root in the container.

Examples:
  # Build with default sandbox base
  coi build custom my-rust-image --script build-rust.sh

  # Build with custom base
  coi build custom my-image --base coi-sandbox --script setup.sh

  # Build with privileged base
  coi build custom my-priv-image --privileged --script setup.sh`,
	Args: cobra.ExactArgs(1),
	RunE: buildCustomCommand,
}

func init() {
	buildCmd.Flags().BoolVar(&buildForce, "force", false, "Force rebuild even if image exists")

	// Custom build flags
	buildCustomCmd.Flags().String("script", "", "Path to build script (required)")
	buildCustomCmd.Flags().String("base", "", "Base image to build from (default: coi-sandbox)")
	buildCustomCmd.Flags().Bool("privileged", false, "Use privileged base if --base not specified")
	buildCustomCmd.Flags().BoolVar(&buildForce, "force", false, "Force rebuild even if image exists")
	buildCustomCmd.MarkFlagRequired("script")

	buildCmd.AddCommand(buildCustomCmd)
}

func buildCommand(cmd *cobra.Command, args []string) error {
	imageType := args[0]

	// If custom, delegate to custom command
	if imageType == "custom" {
		return fmt.Errorf("use 'coi build custom <name> --script <path>' for custom images")
	}

	// Validate image type
	if imageType != "sandbox" && imageType != "privileged" {
		return fmt.Errorf("invalid image type: %s (must be 'sandbox', 'privileged', or 'custom')", imageType)
	}

	// Check if Incus is available
	if !container.Available() {
		return fmt.Errorf("incus is not available - please install Incus and ensure you're in the incus-admin group")
	}

	// Configure build options
	var opts image.BuildOptions
	opts.Force = buildForce
	opts.ImageType = imageType
	opts.BaseImage = image.BaseImage

	switch imageType {
	case "sandbox":
		opts.AliasName = image.SandboxAlias
		opts.Description = "coi sandbox image (Docker + build tools + sudo)"
	case "privileged":
		opts.AliasName = image.PrivilegedAlias
		opts.Description = "coi privileged image (sandbox + GitHub CLI + SSH)"
	}

	// Logger function
	opts.Logger = func(msg string) {
		fmt.Println(msg)
	}

	// Build the image
	fmt.Printf("Building %s image...\n", imageType)
	builder := image.NewBuilder(opts)
	result := builder.Build()

	if result.Error != nil {
		return fmt.Errorf("build failed: %w", result.Error)
	}

	if result.Skipped {
		fmt.Printf("\nImage already exists. Use --force to rebuild.\n")
		return nil
	}

	fmt.Printf("\n✓ Image '%s' built successfully!\n", opts.AliasName)
	fmt.Printf("  Version: %s\n", result.VersionAlias)
	fmt.Printf("  Fingerprint: %s\n", result.Fingerprint)
	return nil
}

func buildCustomCommand(cmd *cobra.Command, args []string) error {
	imageName := args[0]
	scriptPath, _ := cmd.Flags().GetString("script")
	baseImage, _ := cmd.Flags().GetString("base")
	privileged, _ := cmd.Flags().GetBool("privileged")

	// Check if Incus is available
	if !container.Available() {
		return fmt.Errorf("incus is not available - please install Incus and ensure you're in the incus-admin group")
	}

	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("build script not found: %s", scriptPath)
	}

	// Determine base image
	if baseImage == "" {
		if privileged {
			baseImage = image.PrivilegedAlias
		} else {
			baseImage = image.SandboxAlias
		}
	}

	// Configure build options
	opts := image.BuildOptions{
		ImageType:   "custom",
		AliasName:   imageName,
		Description: fmt.Sprintf("Custom image: %s", imageName),
		BaseImage:   baseImage,
		BuildScript: scriptPath,
		Force:       buildForce,
		Logger: func(msg string) {
			fmt.Fprintf(os.Stderr, "%s\n", msg)
		},
	}

	// Build the image
	fmt.Fprintf(os.Stderr, "Building custom image '%s' from '%s'...\n", imageName, baseImage)
	builder := image.NewBuilder(opts)
	result := builder.Build()

	if result.Error != nil {
		return fmt.Errorf("build failed: %w", result.Error)
	}

	// Output result as JSON (even if skipped)
	output := map[string]interface{}{
		"alias":   imageName,
		"skipped": result.Skipped,
	}

	if !result.Skipped {
		output["fingerprint"] = result.Fingerprint
	} else {
		fmt.Fprintf(os.Stderr, "\nImage already exists. Use --force to rebuild.\n")
	}

	jsonOutput, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonOutput))

	return nil
}
