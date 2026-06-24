package model

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/extensions"
)

// LocalModel is a special model name that indicates that the model is local and with or without api_key.
const LocalModel = "local-model"

func NewModelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model [model_name]",
		Short: "Show or change the default model",
		Long: `Show or change the default model configuration.

If no argument is provided, shows the current default model.
If a model name is provided, sets it as the default model.

Examples:
  jameclaw model                    # Show current default model
  jameclaw model gpt-5.2           # Set gpt-5.2 as default
  jameclaw model claude-sonnet-4.6 # Set claude-sonnet-4.6 as default
  jameclaw model local-model       # Set local VLLM server as default

Note: 'local-model' is a special value for using a local VLLM server
(running at localhost:8000 by default) which does not require an API key.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := internal.GetConfigPath()

			// Load current config
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if len(args) == 0 {
				// Show current default model
				showCurrentModel(cfg)
				return nil
			}

			// Set new default model
			modelName := args[0]
			return setDefaultModel(configPath, cfg, modelName)
		},
	}

	cmd.AddCommand(newProvidersCommand(), newAddCommand())
	return cmd
}

func newProvidersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List provider catalog entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(internal.GetConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			for _, provider := range extensions.ProviderCatalog(cfg) {
				status := "available"
				if provider.Configured {
					status = "configured"
				}
				if provider.Default {
					status = "default"
				}
				fmt.Printf("%-18s %-12s %s\n", provider.ID, status, provider.Name)
				if len(provider.RecommendedModels) > 0 {
					presets := make([]string, 0, len(provider.RecommendedModels))
					for _, preset := range provider.RecommendedModels {
						presets = append(presets, preset.ID)
					}
					fmt.Printf("  presets: %s\n", strings.Join(presets, ", "))
				}
			}
			return nil
		},
	}
}

func newAddCommand() *cobra.Command {
	var apiKey string
	var modelName string
	var setDefault bool

	cmd := &cobra.Command{
		Use:   "add <provider> <preset>",
		Short: "Add a model from the provider catalog",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := internal.GetConfigPath()
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			_, preset, ok := extensions.FindPreset(args[0], args[1])
			if !ok {
				return fmt.Errorf("unknown provider or preset: %s %s", args[0], args[1])
			}
			modelCfg := preset.ToModelConfig(modelName)
			if apiKey != "" {
				modelCfg.SetAPIKey(apiKey)
			}
			if err := modelCfg.Validate(); err != nil {
				return err
			}

			replaced := false
			for i, existing := range cfg.ModelList {
				if existing != nil && existing.ModelName == modelCfg.ModelName {
					if modelCfg.APIKey() == "" {
						modelCfg.SetAPIKey(existing.APIKey())
					}
					cfg.ModelList[i] = modelCfg
					replaced = true
					break
				}
			}
			if !replaced {
				cfg.ModelList = append(cfg.ModelList, modelCfg)
			}
			if setDefault {
				cfg.Agents.Defaults.ModelName = modelCfg.ModelName
			}
			if err := config.SaveConfig(configPath, cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			action := "Added"
			if replaced {
				action = "Updated"
			}
			fmt.Printf("%s model %s (%s)\n", action, modelCfg.ModelName, modelCfg.Model)
			if setDefault {
				fmt.Printf("Default model set to %s\n", modelCfg.ModelName)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for the selected provider")
	cmd.Flags().StringVar(&modelName, "name", "", "Model alias to store in model_list")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set the added model as the default")
	return cmd
}

func showCurrentModel(cfg *config.Config) {
	defaultModel := cfg.Agents.Defaults.ModelName

	if defaultModel == "" {
		fmt.Println("No default model is currently set.")
		fmt.Println("\nAvailable models in your config:")
		listAvailableModels(cfg)
	} else {
		fmt.Printf("Current default model: %s\n", defaultModel)
		fmt.Println("\nAvailable models in your config:")
		listAvailableModels(cfg)
	}
}

func listAvailableModels(cfg *config.Config) {
	if len(cfg.ModelList) == 0 {
		fmt.Println("  No models configured in model_list")
		return
	}

	defaultModel := cfg.Agents.Defaults.ModelName

	for _, model := range cfg.ModelList {
		marker := "  "
		if model.ModelName == defaultModel {
			marker = "> "
		}
		if model.APIKey() == "" {
			continue
		}
		fmt.Printf("%s- %s (%s)\n", marker, model.ModelName, model.Model)
	}
}

func setDefaultModel(configPath string, cfg *config.Config, modelName string) error {
	// Validate that the model exists in model_list
	modelFound := false
	for _, model := range cfg.ModelList {
		if model.APIKey() != "" && model.ModelName == modelName {
			modelFound = true
			break
		}
	}

	if !modelFound && modelName != LocalModel {
		return fmt.Errorf("cannot found model '%s' in config", modelName)
	}

	// Update the default model
	// Clear old model field and set new model_name
	oldModel := cfg.Agents.Defaults.ModelName

	cfg.Agents.Defaults.ModelName = modelName

	// Save config back to file
	if err := config.SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Default model changed from '%s' to '%s'\n",
		formatModelName(oldModel), modelName)
	fmt.Println("\nThe new default model will be used for all agent interactions.")

	return nil
}

func formatModelName(name string) string {
	if name == "" {
		return "(none)"
	}
	return name
}
