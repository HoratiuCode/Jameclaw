package extensions

import (
	"slices"
	"strings"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/providers"
)

type ProviderAuthMethod struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type ModelPreset struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ModelName      string `json:"model_name"`
	Model          string `json:"model"`
	APIBase        string `json:"api_base,omitempty"`
	RequiresAPIKey bool   `json:"requires_api_key"`
	KeyLabel       string `json:"key_label,omitempty"`
	RequestTimeout int    `json:"request_timeout,omitempty"`
	ThinkingLevel  string `json:"thinking_level,omitempty"`
	Description    string `json:"description,omitempty"`
}

func (p ModelPreset) ToModelConfig(modelName string) *config.ModelConfig {
	if strings.TrimSpace(modelName) == "" {
		modelName = p.ModelName
	}
	return &config.ModelConfig{
		ModelName:      modelName,
		Model:          p.Model,
		APIBase:        p.APIBase,
		RequestTimeout: p.RequestTimeout,
		ThinkingLevel:  p.ThinkingLevel,
	}
}

type ProviderDescriptor struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Category          string               `json:"category"`
	Description       string               `json:"description"`
	DocsURL           string               `json:"docs_url,omitempty"`
	Protocols         []string             `json:"protocols"`
	DefaultAPIBase    string               `json:"default_api_base,omitempty"`
	RequiresAPIKey    bool                 `json:"requires_api_key"`
	KeyLabel          string               `json:"key_label,omitempty"`
	AuthMethods       []ProviderAuthMethod `json:"auth_methods,omitempty"`
	RecommendedModels []ModelPreset        `json:"recommended_models"`
	SetupHint         string               `json:"setup_hint,omitempty"`
	LocalRuntimeHint  string               `json:"local_runtime_hint,omitempty"`
	Configured        bool                 `json:"configured"`
	Default           bool                 `json:"default"`
	ConfiguredModels  []string             `json:"configured_models,omitempty"`
}

type Catalog struct {
	Providers []ProviderDescriptor `json:"providers"`
	Tools     []CatalogItem        `json:"tools,omitempty"`
	Channels  []CatalogItem        `json:"channels,omitempty"`
}

type CatalogItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	ConfigKey   string `json:"config_key,omitempty"`
}

func ProviderCatalog(cfg *config.Config) []ProviderDescriptor {
	base := providerDescriptors()
	if cfg == nil {
		return base
	}

	defaultModel := cfg.Agents.Defaults.GetModelName()
	for i := range base {
		for _, modelCfg := range cfg.ModelList {
			if modelCfg == nil {
				continue
			}
			ref := providers.ParseModelRef(modelCfg.Model, "")
			if ref == nil {
				continue
			}
			if !slices.Contains(base[i].Protocols, ref.Provider) {
				continue
			}
			base[i].Configured = true
			if modelCfg.ModelName == defaultModel {
				base[i].Default = true
			}
			if !slices.Contains(base[i].ConfiguredModels, modelCfg.ModelName) {
				base[i].ConfiguredModels = append(base[i].ConfiguredModels, modelCfg.ModelName)
			}
		}
	}
	return base
}

func FindProvider(id string) (ProviderDescriptor, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, provider := range providerDescriptors() {
		if provider.ID == id || slices.Contains(provider.Protocols, id) {
			return provider, true
		}
	}
	return ProviderDescriptor{}, false
}

func FindPreset(providerID, presetID string) (ProviderDescriptor, ModelPreset, bool) {
	provider, ok := FindProvider(providerID)
	if !ok {
		return ProviderDescriptor{}, ModelPreset{}, false
	}
	presetID = strings.ToLower(strings.TrimSpace(presetID))
	for _, preset := range provider.RecommendedModels {
		if preset.ID == presetID || strings.EqualFold(preset.ModelName, presetID) {
			return provider, preset, true
		}
	}
	return ProviderDescriptor{}, ModelPreset{}, false
}

func providerDescriptors() []ProviderDescriptor {
	return []ProviderDescriptor{
		httpProvider("openai", "OpenAI", "https://platform.openai.com/api-keys", "https://api.openai.com/v1", "gpt-5.4", "openai/gpt-5.4"),
		{
			ID:             "anthropic",
			Name:           "Anthropic",
			Category:       "frontier",
			Description:    "Claude models through Anthropic API or OAuth/token auth.",
			DocsURL:        "https://console.anthropic.com/settings/keys",
			Protocols:      []string{"anthropic", "anthropic-messages", "coding-plan-anthropic", "alibaba-coding-anthropic"},
			DefaultAPIBase: "https://api.anthropic.com/v1",
			RequiresAPIKey: true,
			KeyLabel:       "Anthropic API key",
			AuthMethods: []ProviderAuthMethod{
				{ID: "api_key", Label: "API key"},
				{ID: "oauth", Label: "OAuth/token"},
			},
			RecommendedModels: []ModelPreset{{
				ID:             "claude-sonnet-4-6",
				Name:           "Claude Sonnet 4.6",
				ModelName:      "claude-sonnet-4.6",
				Model:          "anthropic/claude-sonnet-4.6",
				APIBase:        "https://api.anthropic.com/v1",
				RequiresAPIKey: true,
				KeyLabel:       "Anthropic API key",
				ThinkingLevel:  "high",
			}},
			SetupHint: "Paste an Anthropic API key, or use Jameclaw auth for token-based Claude access.",
		},
		httpProvider("openrouter", "OpenRouter", "https://openrouter.ai/keys", "https://openrouter.ai/api/v1", "openrouter-auto", "openrouter/auto"),
		httpProviderWithProtocols("azure", "Azure OpenAI", "https://learn.microsoft.com/azure/ai-services/openai/", "https://your-resource.openai.azure.com", []string{"azure", "azure-openai"}, "azure-gpt", "azure/gpt-4o"),
		httpProvider("bedrock", "Amazon Bedrock", "https://aws.amazon.com/bedrock/", "us-east-1", "bedrock-claude", "bedrock/anthropic.claude-3-5-sonnet-20241022-v2:0"),
		httpProvider("gemini", "Google Gemini", "https://ai.google.dev/", "https://generativelanguage.googleapis.com/v1beta", "gemini-2.0-flash", "gemini/gemini-2.0-flash-exp"),
		httpProvider("deepseek", "DeepSeek", "https://platform.deepseek.com/", "https://api.deepseek.com/v1", "deepseek-chat", "deepseek/deepseek-chat"),
		httpProvider("qwen", "Qwen", "https://dashscope.console.aliyun.com/apiKey", "https://dashscope.aliyuncs.com/compatible-mode/v1", "qwen-plus", "qwen/qwen-plus"),
		httpProvider("moonshot", "Moonshot / Kimi", "https://platform.moonshot.cn/console/api-keys", "https://api.moonshot.cn/v1", "moonshot-v1-8k", "moonshot/moonshot-v1-8k"),
		httpProvider("groq", "Groq", "https://console.groq.com/keys", "https://api.groq.com/openai/v1", "llama-3.3-70b", "groq/llama-3.3-70b-versatile"),
		httpProvider("mistral", "Mistral AI", "https://console.mistral.ai/api-keys", "https://api.mistral.ai/v1", "mistral-large", "mistral/mistral-large-latest"),
		httpProvider("minimax", "MiniMax", "https://platform.minimaxi.com/", "https://api.minimaxi.com/v1", "minimax-chat", "minimax/MiniMax-M1"),
		httpProvider("zhipu", "Zhipu AI", "https://open.bigmodel.cn/usercenter/apikeys", "https://open.bigmodel.cn/api/paas/v4", "glm-4.7", "zhipu/glm-4.7"),
		httpProvider("nvidia", "NVIDIA", "https://build.nvidia.com/", "https://integrate.api.nvidia.com/v1", "nvidia-llama", "nvidia/meta/llama-3.1-405b-instruct"),
		httpProvider("cerebras", "Cerebras", "https://cloud.cerebras.ai/", "https://api.cerebras.ai/v1", "cerebras-llama", "cerebras/llama-4-scout-17b-16e-instruct"),
		httpProvider("novita", "Novita", "https://novita.ai/", "https://api.novita.ai/openai", "novita-model", "novita/meta-llama/llama-3.1-8b-instruct"),
		httpProvider("litellm", "LiteLLM", "https://docs.litellm.ai/", "http://localhost:4000/v1", "litellm-local", "litellm/gpt-4o"),
		httpProvider("volcengine", "Volcengine", "https://console.volcengine.com/ark/", "https://ark.cn-beijing.volces.com/api/v3", "volcengine-model", "volcengine/doubao-seed-1-6"),
		httpProvider("shengsuanyun", "ShengsuanYun", "https://shengsuanyun.com/", "https://router.shengsuanyun.com/api/v1", "shengsuanyun-model", "shengsuanyun/default"),
		httpProviderWithProtocols("qwen-intl", "Qwen International", "https://www.alibabacloud.com/help/en/model-studio/", "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", []string{"qwen-intl", "qwen-international", "dashscope-intl"}, "qwen-intl-plus", "qwen-intl/qwen-plus"),
		httpProviderWithProtocols("qwen-us", "Qwen US", "https://www.alibabacloud.com/help/en/model-studio/", "https://dashscope-us.aliyuncs.com/compatible-mode/v1", []string{"qwen-us", "dashscope-us"}, "qwen-us-plus", "qwen-us/qwen-plus"),
		httpProviderWithProtocols("coding-plan", "Alibaba Coding Plan", "https://www.alibabacloud.com/help/en/model-studio/", "https://coding-intl.dashscope.aliyuncs.com/v1", []string{"coding-plan", "alibaba-coding", "qwen-coding"}, "coding-plan", "coding-plan/qwen-coder"),
		httpProvider("avian", "Avian", "https://avian.io/", "https://api.avian.io/v1", "avian-model", "avian/default"),
		httpProvider("longcat", "LongCat", "https://longcat.chat/", "https://api.longcat.chat/openai", "longcat-model", "longcat/default"),
		httpProvider("modelscope", "ModelScope", "https://modelscope.cn/", "https://api-inference.modelscope.cn/v1", "modelscope-model", "modelscope/default"),
		httpProvider("nous", "Nous", "https://nousresearch.com/", "https://inference-api.nousresearch.com/v1", "nous-model", "nous/default"),
		httpProvider("vivgrid", "Vivgrid", "https://vivgrid.com/", "https://api.vivgrid.com/v1", "vivgrid-model", "vivgrid/default"),
		localProvider("ollama", "Ollama", "http://localhost:11434/v1", "llama3", "ollama/llama3", "Install Ollama locally and pull the model before using it."),
		localProvider("vllm", "vLLM", "http://localhost:8000/v1", "local-model", "vllm/local-model", "Start a vLLM OpenAI-compatible server on localhost:8000."),
		localProviderWithProtocols("claude-cli", "Claude CLI", []string{"claude-cli", "claudecli"}, "", "claude-cli", "claude-cli/sonnet", "Install and authenticate the Claude CLI."),
		localProviderWithProtocols("codex-cli", "Codex CLI", []string{"codex-cli", "codexcli"}, "", "codex-cli", "codex-cli/gpt-5.4", "Install and authenticate the Codex CLI."),
		localProviderWithProtocols("grok-build", "Grok Build", []string{"grok-cli", "grokcli", "grok-build"}, "", "grok-build", "grok-build/default", "Use the signed-in Grok Build CLI installed on this Mac."),
		localProvider("antigravity", "Google Code Assist", "", "antigravity", "antigravity/default", "Authenticate Google Code Assist / Antigravity locally."),
		{
			ID:             "github-copilot",
			Name:           "GitHub Copilot",
			Category:       "local-bridge",
			Description:    "Connect to a local GitHub Copilot bridge.",
			Protocols:      []string{"github-copilot", "copilot"},
			DefaultAPIBase: "localhost:4321",
			RequiresAPIKey: false,
			AuthMethods:    []ProviderAuthMethod{{ID: "grpc", Label: "gRPC bridge"}},
			RecommendedModels: []ModelPreset{{
				ID:        "copilot",
				Name:      "GitHub Copilot",
				ModelName: "github-copilot",
				Model:     "github-copilot/default",
				APIBase:   "localhost:4321",
			}},
			LocalRuntimeHint: "Run the local Copilot bridge before selecting this model.",
		},
	}
}

func httpProvider(id, name, docsURL, apiBase, modelName, model string) ProviderDescriptor {
	return httpProviderWithProtocols(id, name, docsURL, apiBase, []string{id}, modelName, model)
}

func httpProviderWithProtocols(id, name, docsURL, apiBase string, protocols []string, modelName, model string) ProviderDescriptor {
	return ProviderDescriptor{
		ID:             id,
		Name:           name,
		Category:       "openai-compatible",
		Description:    name + " models through an HTTP API.",
		DocsURL:        docsURL,
		Protocols:      protocols,
		DefaultAPIBase: apiBase,
		RequiresAPIKey: true,
		KeyLabel:       name + " API key",
		AuthMethods:    []ProviderAuthMethod{{ID: "api_key", Label: "API key"}},
		RecommendedModels: []ModelPreset{{
			ID:             modelName,
			Name:           modelName,
			ModelName:      modelName,
			Model:          model,
			APIBase:        apiBase,
			RequiresAPIKey: true,
			KeyLabel:       name + " API key",
			RequestTimeout: 60,
		}},
		SetupHint: "Paste an API key and use the default API base unless you run a compatible proxy.",
	}
}

func localProvider(id, name, apiBase, modelName, model, hint string) ProviderDescriptor {
	return localProviderWithProtocols(id, name, []string{id}, apiBase, modelName, model, hint)
}

func localProviderWithProtocols(id, name string, protocols []string, apiBase, modelName, model, hint string) ProviderDescriptor {
	return ProviderDescriptor{
		ID:               id,
		Name:             name,
		Category:         "local",
		Description:      name + " local runtime provider.",
		Protocols:        protocols,
		DefaultAPIBase:   apiBase,
		RequiresAPIKey:   false,
		AuthMethods:      []ProviderAuthMethod{{ID: "local", Label: "Local runtime"}},
		LocalRuntimeHint: hint,
		RecommendedModels: []ModelPreset{{
			ID:        modelName,
			Name:      modelName,
			ModelName: modelName,
			Model:     model,
			APIBase:   apiBase,
		}},
	}
}
