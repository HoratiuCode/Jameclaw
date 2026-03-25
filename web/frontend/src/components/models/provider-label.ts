const PROVIDER_LABELS: Record<string, string> = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  gemini: "Google Gemini",
  deepseek: "DeepSeek",
  qwen: "Qwen",
  moonshot: "Moonshot",
  groq: "Groq",
  openrouter: "OpenRouter",
  nvidia: "NVIDIA",
  cerebras: "Cerebras",
  volcengine: "Volcengine",
  shengsuanyun: "ShengsuanYun",
  antigravity: "Google Code Assist",
  "github-copilot": "GitHub Copilot",
  ollama: "Ollama (local)",
  mistral: "Mistral AI",
  avian: "Avian",
  vllm: "VLLM (local)",
  zhipu: "Zhipu AI",
}

export function getProviderKey(model: string): string {
  return model.split("/")[0]
}

export function getProviderLabel(model: string): string {
  const prefix = getProviderKey(model)
  const labels: Record<string, string> = {
    ...PROVIDER_LABELS,
  }
  return labels[prefix] ?? prefix
}
