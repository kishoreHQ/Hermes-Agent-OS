// Package providercfg manages UI-driven provider configs and popular LLM templates.
package providercfg

import "github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"

// Template is a pre-filled popular LLM provider recipe (OpenAI-compatible wire format).
type Template struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Driver      string           `json:"driver"` // openai-compat | echo-provider
	BaseURL     string           `json:"baseUrl"`
	Local       bool             `json:"local"`
	CostTier    types.CostTier   `json:"costTier"`
	DefaultModel string          `json:"defaultModel,omitempty"`
	DocsURL     string           `json:"docsUrl,omitempty"`
	NeedsAPIKey bool             `json:"needsApiKey"`
	// SuggestedModels for UI (operator can edit; discovery may replace).
	SuggestedModels []string `json:"suggestedModels,omitempty"`
	Category        string   `json:"category,omitempty"` // cloud | local | gateway
}

// Templates returns popular provider base configs (OpenAI Chat Completions compatible).
// Official Anthropic/Google native APIs differ; use OpenRouter/compat gateways for those, or native plugins later.
func Templates() []Template {
	return []Template{
		{
			// Kimchi Inference (Cast AI) — same OpenAI-compat endpoint as Cursor/OpenCode docs.
			// Docs: https://docs.kimchi.dev/docs/cursor · https://docs.kimchi.dev/docs/inference-quickstart
			// Base URL: https://llm.kimchi.dev/openai/v1 · Key: app.kimchi.dev/settings
			ID: "kimchi", Name: "Kimchi", Driver: "openai-compat",
			Description: "Kimchi Inference — open-source models (Kimi K2.7, MiniMax M3) via OpenAI-compatible API",
			BaseURL: "https://llm.kimchi.dev/openai/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "kimi-k2.7", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.kimchi.dev/docs/cursor",
			SuggestedModels: []string{"kimi-k2.7", "kimi-k2.5", "minimax-m3", "minimax-m2.7", "nemotron-3-ultra-fp4", "deepseek-v4-flash", "glm-5.2-fp8", "qwen3-coder-next-fp8"},
		},
		{
			ID: "openai", Name: "OpenAI", Driver: "openai-compat",
			Description: "OpenAI official API (Chat Completions)",
			BaseURL: "https://api.openai.com/v1", Local: false, CostTier: types.TierStandard,
			DefaultModel: "gpt-4o-mini", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://platform.openai.com/docs/api-reference",
			SuggestedModels: []string{"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "o3-mini"},
		},
		{
			ID: "openrouter", Name: "OpenRouter", Driver: "openai-compat",
			Description: "Multi-model gateway (OpenAI-compatible)",
			BaseURL: "https://openrouter.ai/api/v1", Local: false, CostTier: types.TierStandard,
			DefaultModel: "openai/gpt-4o-mini", NeedsAPIKey: true, Category: "gateway",
			DocsURL: "https://openrouter.ai/docs",
			SuggestedModels: []string{"openai/gpt-4o-mini", "anthropic/claude-sonnet-4", "google/gemini-2.0-flash-001", "meta-llama/llama-3.3-70b-instruct"},
		},
		{
			ID: "groq", Name: "Groq", Driver: "openai-compat",
			Description: "Groq fast inference (OpenAI-compatible)",
			BaseURL: "https://api.groq.com/openai/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "llama-3.3-70b-versatile", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://console.groq.com/docs",
			SuggestedModels: []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768", "gemma2-9b-it"},
		},
		{
			ID: "together", Name: "Together AI", Driver: "openai-compat",
			Description: "Together OpenAI-compatible API",
			BaseURL: "https://api.together.xyz/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.together.ai",
		},
		{
			ID: "fireworks", Name: "Fireworks AI", Driver: "openai-compat",
			Description: "Fireworks OpenAI-compatible API",
			BaseURL: "https://api.fireworks.ai/inference/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "accounts/fireworks/models/llama-v3p1-70b-instruct", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.fireworks.ai",
		},
		{
			ID: "deepseek", Name: "DeepSeek", Driver: "openai-compat",
			Description: "DeepSeek OpenAI-compatible API",
			BaseURL: "https://api.deepseek.com/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "deepseek-chat", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://api-docs.deepseek.com",
			SuggestedModels: []string{"deepseek-chat", "deepseek-reasoner"},
		},
		{
			ID: "mistral", Name: "Mistral AI", Driver: "openai-compat",
			Description: "Mistral OpenAI-compatible API",
			BaseURL: "https://api.mistral.ai/v1", Local: false, CostTier: types.TierStandard,
			DefaultModel: "mistral-small-latest", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.mistral.ai",
			SuggestedModels: []string{"mistral-small-latest", "mistral-large-latest", "codestral-latest"},
		},
		{
			ID: "xai", Name: "xAI (Grok)", Driver: "openai-compat",
			Description: "xAI Grok OpenAI-compatible API",
			BaseURL: "https://api.x.ai/v1", Local: false, CostTier: types.TierStandard,
			DefaultModel: "grok-3", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.x.ai",
			SuggestedModels: []string{"grok-3", "grok-3-mini", "grok-2-latest", "grok-2-vision-latest"},
		},
		{
			ID: "google-openai", Name: "Google Gemini (OpenAI compat)", Driver: "openai-compat",
			Description: "Gemini via OpenAI-compatible endpoint",
			BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", Local: false, CostTier: types.TierStandard,
			DefaultModel: "gemini-2.0-flash", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://ai.google.dev/gemini-api/docs/openai",
			SuggestedModels: []string{"gemini-2.0-flash", "gemini-2.5-pro", "gemini-1.5-pro", "gemini-1.5-flash"},
		},
		{
			ID: "azure-openai", Name: "Azure OpenAI", Driver: "openai-compat",
			Description: "Azure OpenAI — set baseURL to your resource (.../openai/v1)",
			BaseURL: "https://YOUR_RESOURCE.openai.azure.com/openai/v1", Local: false, CostTier: types.TierStandard,
			DefaultModel: "gpt-4o-mini", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://learn.microsoft.com/azure/ai-services/openai/",
		},
		{
			ID: "perplexity", Name: "Perplexity", Driver: "openai-compat",
			Description: "Perplexity Sonar OpenAI-compatible API",
			BaseURL: "https://api.perplexity.ai", Local: false, CostTier: types.TierStandard,
			DefaultModel: "sonar", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.perplexity.ai",
			SuggestedModels: []string{"sonar", "sonar-pro", "sonar-reasoning"},
		},
		{
			ID: "cerebras", Name: "Cerebras", Driver: "openai-compat",
			Description: "Cerebras Inference OpenAI-compatible API",
			BaseURL: "https://api.cerebras.ai/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "llama-3.3-70b", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://inference-docs.cerebras.ai",
			SuggestedModels: []string{"llama-3.3-70b", "llama3.1-8b"},
		},
		{
			ID: "sambanova", Name: "SambaNova", Driver: "openai-compat",
			Description: "SambaNova Cloud OpenAI-compatible API",
			BaseURL: "https://api.sambanova.ai/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "Meta-Llama-3.3-70B-Instruct", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.sambanova.ai",
		},
		{
			ID: "huggingface", Name: "Hugging Face Inference", Driver: "openai-compat",
			Description: "HF router OpenAI-compatible endpoint",
			BaseURL: "https://router.huggingface.co/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "meta-llama/Llama-3.3-70B-Instruct", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://huggingface.co/docs/api-inference",
		},
		{
			ID: "cloudflare", Name: "Cloudflare Workers AI", Driver: "openai-compat",
			Description: "Workers AI OpenAI-compatible — replace ACCOUNT_ID in URL",
			BaseURL: "https://api.cloudflare.com/client/v4/accounts/ACCOUNT_ID/ai/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "@cf/meta/llama-3.1-8b-instruct", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://developers.cloudflare.com/workers-ai/",
		},
		{
			ID: "nvidia", Name: "NVIDIA NIM", Driver: "openai-compat",
			Description: "NVIDIA build.nvidia.com OpenAI-compatible API",
			BaseURL: "https://integrate.api.nvidia.com/v1", Local: false, CostTier: types.TierStandard,
			DefaultModel: "meta/llama-3.1-70b-instruct", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://docs.api.nvidia.com",
		},
		{
			ID: "moonshot", Name: "Moonshot (Kimi)", Driver: "openai-compat",
			Description: "Moonshot AI OpenAI-compatible API",
			BaseURL: "https://api.moonshot.ai/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "moonshot-v1-8k", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://platform.moonshot.ai/docs",
			SuggestedModels: []string{"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k", "kimi-latest"},
		},
		{
			ID: "qwen", Name: "Alibaba Qwen (DashScope)", Driver: "openai-compat",
			Description: "DashScope OpenAI-compatible endpoint (intl)",
			BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", Local: false, CostTier: types.TierBudget,
			DefaultModel: "qwen-plus", NeedsAPIKey: true, Category: "cloud",
			DocsURL: "https://www.alibabacloud.com/help/model-studio",
			SuggestedModels: []string{"qwen-plus", "qwen-turbo", "qwen-max", "qwen-long"},
		},
		{
			ID: "anthropic-via-openrouter", Name: "Anthropic via OpenRouter", Driver: "openai-compat",
			Description: "Claude models through OpenRouter (native Anthropic Messages API is a future plugin)",
			BaseURL: "https://openrouter.ai/api/v1", Local: false, CostTier: types.TierPremium,
			DefaultModel: "anthropic/claude-sonnet-4", NeedsAPIKey: true, Category: "gateway",
			SuggestedModels: []string{"anthropic/claude-sonnet-4", "anthropic/claude-3.5-sonnet", "anthropic/claude-3-haiku"},
		},
		{
			ID: "lite-llm", Name: "LiteLLM proxy", Driver: "openai-compat",
			Description: "Self-hosted LiteLLM multi-provider gateway",
			BaseURL: "http://127.0.0.1:4000/v1", Local: true, CostTier: types.TierStandard,
			DefaultModel: "gpt-4o-mini", NeedsAPIKey: true, Category: "gateway",
			DocsURL: "https://docs.litellm.ai",
		},
		{
			ID: "ollama", Name: "Ollama (local)", Driver: "openai-compat",
			Description: "Local Ollama OpenAI-compatible server",
			BaseURL: "http://127.0.0.1:11434/v1", Local: true, CostTier: types.TierFreeLocal,
			DefaultModel: "llama3.2", NeedsAPIKey: false, Category: "local",
			DocsURL: "https://github.com/ollama/ollama/blob/main/docs/openai.md",
			SuggestedModels: []string{"llama3.2", "llama3.1", "codellama", "mistral", "qwen2.5"},
		},
		{
			ID: "lmstudio", Name: "LM Studio (local)", Driver: "openai-compat",
			Description: "LM Studio local server",
			BaseURL: "http://127.0.0.1:1234/v1", Local: true, CostTier: types.TierFreeLocal,
			DefaultModel: "local-model", NeedsAPIKey: false, Category: "local",
			DocsURL: "https://lmstudio.ai/docs",
		},
		{
			ID: "vllm", Name: "vLLM (local/self-host)", Driver: "openai-compat",
			Description: "vLLM OpenAI-compatible server",
			BaseURL: "http://127.0.0.1:8000/v1", Local: true, CostTier: types.TierFreeLocal,
			DefaultModel: "default", NeedsAPIKey: false, Category: "local",
			DocsURL: "https://docs.vllm.ai",
		},
		{
			ID: "localai", Name: "LocalAI", Driver: "openai-compat",
			Description: "LocalAI OpenAI-compatible server",
			BaseURL: "http://127.0.0.1:8080/v1", Local: true, CostTier: types.TierFreeLocal,
			DefaultModel: "gpt-4", NeedsAPIKey: false, Category: "local",
			DocsURL: "https://localai.io",
		},
		{
			ID: "custom", Name: "Custom OpenAI-compatible", Driver: "openai-compat",
			Description: "Any Chat Completions-compatible endpoint — fill your base URL",
			BaseURL: "https://your-endpoint.example.com/v1", Local: false, CostTier: types.TierStandard,
			DefaultModel: "default", NeedsAPIKey: true, Category: "custom",
		},
		{
			ID: "echo", Name: "Echo (test)", Driver: "echo-provider",
			Description: "Deterministic in-process provider for tests (no network)",
			BaseURL: "", Local: true, CostTier: types.TierFreeLocal,
			DefaultModel: "echo-1", NeedsAPIKey: false, Category: "local",
		},
	}
}

// TemplateByID finds a template.
func TemplateByID(id string) (Template, bool) {
	for _, t := range Templates() {
		if t.ID == id {
			return t, true
		}
	}
	return Template{}, false
}
