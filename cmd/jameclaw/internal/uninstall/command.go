package uninstall

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/credential"
	"github.com/sipeed/jameclaw/web/backend/launcherconfig"
)

type options struct {
	yes bool
}

type resetPlan struct {
	HomePath           string
	ConfigPath         string
	LauncherConfigPath string
	SSHKeyPath         string
}

var uninstallInput io.Reader = os.Stdin
var uninstallOutput io.Writer = os.Stdout

func NewUninstallCommand() *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"reset"},
		Short:   "Remove local jameclaw data and reset first-run setup",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func run(opts options) error {
	plan, err := buildResetPlan()
	if err != nil {
		return err
	}
	if err := validateHomePath(plan.HomePath); err != nil {
		return err
	}

	if !opts.yes {
		confirmed, err := promptConfirm(plan)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(uninstallOutput, "JameClaw uninstall canceled.")
			return nil
		}
	}

	removedCount := 0
	for _, path := range []string{plan.ConfigPath, plan.LauncherConfigPath} {
		removed, err := removeFileIfExists(path)
		if err != nil {
			return err
		}
		if removed {
			removedCount++
		}
	}

	removedHome, err := removeDirIfExists(plan.HomePath)
	if err != nil {
		return err
	}
	if removedHome {
		removedCount++
	}

	for _, path := range []string{plan.SSHKeyPath, plan.SSHKeyPath + ".pub"} {
		removed, err := removeFileIfExists(path)
		if err != nil {
			return err
		}
		if removed {
			removedCount++
		}
	}

	if removedCount == 0 {
		fmt.Fprintln(uninstallOutput, "No local JameClaw state was found.")
		fmt.Fprintln(uninstallOutput, "The next setup can start with `jameclaw install`.")
		return nil
	}

	fmt.Fprintln(uninstallOutput, "JameClaw local state removed.")
	fmt.Fprintln(uninstallOutput, "To start fresh again, run `jameclaw install`.")
	return nil
}

func buildResetPlan() (resetPlan, error) {
	configPath := internal.GetConfigPath()
	sshKeyPath, err := credential.DefaultSSHKeyPath()
	if err != nil {
		return resetPlan{}, fmt.Errorf("resolve SSH key path: %w", err)
	}

	return resetPlan{
		HomePath:           internal.GetJameclawHome(),
		ConfigPath:         configPath,
		LauncherConfigPath: launcherconfig.PathForAppConfig(configPath),
		SSHKeyPath:         sshKeyPath,
	}, nil
}

func validateHomePath(path string) error {
	cleaned := filepath.Clean(path)
	if cleaned == "" || cleaned == "." || cleaned == string(os.PathSeparator) {
		return fmt.Errorf("refusing to remove unsafe JameClaw home path %q", path)
	}

	userHome, err := os.UserHomeDir()
	if err == nil && cleaned == filepath.Clean(userHome) {
		return fmt.Errorf("refusing to remove home directory %q", path)
	}

	return nil
}

func promptConfirm(plan resetPlan) (bool, error) {
	reader := bufio.NewReader(uninstallInput)

	fmt.Fprintln(uninstallOutput, "This removes local JameClaw data and resets first-run setup.")
	fmt.Fprintf(uninstallOutput, "Home: %s\n", plan.HomePath)
	fmt.Fprintf(uninstallOutput, "Config: %s\n", plan.ConfigPath)
	fmt.Fprintf(uninstallOutput, "Launcher config: %s\n", plan.LauncherConfigPath)
	fmt.Fprintf(uninstallOutput, "SSH key: %s\n", plan.SSHKeyPath)
	fmt.Fprint(uninstallOutput, "Continue? [y/N]: ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func removeFileIfExists(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("remove %s: %w", path, err)
	}
	return true, nil
}

func removeDirIfExists(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("check %s: %w", path, err)
	}
	if err := os.RemoveAll(path); err != nil {
		return false, fmt.Errorf("remove %s: %w", path, err)
	}
	return true, nil
}
