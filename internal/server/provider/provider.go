// Package provider defines the model provider configuration layer for the
// Aster web server. It supports preset mainstream providers (all OpenAI-compatible
// protocol) and arbitrary self-hosted / intranet endpoints.
package provider

import (
	"fmt"
	"os"
	"strings"

	"aster/internal/ai"
	"aster/internal/ai/anthropic"
	"aster/internal/ai/openai"
)

// Protocol identifies the wire protocol a provider speaks.
type Protocol string

const (
	ProtocolOpenAI    Protocol = "openai"    // OpenAI-compatible /chat/completions
	ProtocolAnthropic Protocol = "anthropic" // Anthropic Messages API
)

// Preset is a built-in provider template. Users pick a preset by ID and only
// need to supply an API key (and optionally override the model). Self-hosted
// intranet deployments use the "custom" preset and supply their own base URL.
type Preset struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Protocol        Protocol `json:"protocol"`
	BaseURL         string   `json:"base_url"`
	DefaultModel    string   `json:"default_model"`
	RequiresBaseURL bool     `json:"requires_base_url"`
	RequiresAPIKey  bool     `json:"requires_api_key"`
}

// Presets returns the built-in provider catalog. All mainstream Chinese and
// international providers below speak the OpenAI-compatible protocol, so they
// share the same client implementation and differ only by base URL + model.
func Presets() []Preset {
	return []Preset{
		{
			ID: "openai", Label: "OpenAI", Protocol: ProtocolOpenAI,
			BaseURL: "https://api.openai.com/v1/chat/completions",
			DefaultModel: "gpt-4o", RequiresAPIKey: true,
		},
		{
			ID: "deepseek", Label: "DeepSeek", Protocol: ProtocolOpenAI,
			BaseURL: "https://api.deepseek.com/v1/chat/completions",
			DefaultModel: "deepseek-chat", RequiresAPIKey: true,
		},
		{
			ID: "zhipu", Label: "智谱 GLM", Protocol: ProtocolOpenAI,
			BaseURL: "https://open.bigmodel.cn/api/paas/v4/chat/completions",
			DefaultModel: "glm-4-plus", RequiresAPIKey: true,
		},
		{
			ID: "moonshot", Label: "月之暗面 Kimi", Protocol: ProtocolOpenAI,
			BaseURL: "https://api.moonshot.cn/v1/chat/completions",
			DefaultModel: "moonshot-v1-32k", RequiresAPIKey: true,
		},
		{
			ID: "qwen", Label: "阿里通义千问", Protocol: ProtocolOpenAI,
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions",
			DefaultModel: "qwen-max", RequiresAPIKey: true,
		},
		{
			ID: "anthropic", Label: "Anthropic Claude", Protocol: ProtocolAnthropic,
			BaseURL: "https://api.anthropic.com/v1/messages",
			DefaultModel: "claude-3-5-sonnet-20241022", RequiresAPIKey: true,
		},
		{
			ID: "custom", Label: "自定义 / 内网部署", Protocol: ProtocolOpenAI,
			BaseURL: "", DefaultModel: "",
			RequiresBaseURL: true, RequiresAPIKey: false,
		},
	}
}

// PresetByID returns the preset with the given ID, or false if not found.
func PresetByID(id string) (Preset, bool) {
	for _, p := range Presets() {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// Config is a fully-resolved provider configuration ready to build a client.
type Config struct {
	Preset             string   `json:"preset"`
	Protocol           Protocol `json:"protocol"`
	BaseURL            string   `json:"base_url"`
	APIKey             string   `json:"-"` // never serialized
	Model              string   `json:"model"`
	Proxy              string   `json:"proxy,omitempty"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify,omitempty"`
}

// LoadFromEnv builds a Config from environment variables. This is the server's
// primary configuration path for production deployment.
//
//	ASTER_MODEL_PROVIDER   preset id (default "openai"; "custom" for intranet)
//	ASTER_MODEL_BASE_URL   endpoint override (required for custom)
//	ASTER_MODEL_API_KEY    api key
//	ASTER_MODEL_NAME       model id override
//	ASTER_MODEL_PROXY      optional http(s) proxy
//	ASTER_MODEL_INSECURE   "1"/"true" to skip TLS verify (intranet self-signed)
func LoadFromEnv() (*Config, error) {
	presetID := strings.TrimSpace(os.Getenv("ASTER_MODEL_PROVIDER"))
	if presetID == "" {
		presetID = "openai"
	}
	preset, ok := PresetByID(presetID)
	if !ok {
		return nil, fmt.Errorf("unknown model provider preset %q (see GET /api/v1/providers)", presetID)
	}

	cfg := &Config{
		Preset:             preset.ID,
		Protocol:           preset.Protocol,
		BaseURL:            preset.BaseURL,
		APIKey:             os.Getenv("ASTER_MODEL_API_KEY"),
		Model:              preset.DefaultModel,
		Proxy:              os.Getenv("ASTER_MODEL_PROXY"),
		InsecureSkipVerify: boolEnv("ASTER_MODEL_INSECURE"),
	}
	if u := strings.TrimSpace(os.Getenv("ASTER_MODEL_BASE_URL")); u != "" {
		cfg.BaseURL = u
	}
	if m := strings.TrimSpace(os.Getenv("ASTER_MODEL_NAME")); m != "" {
		cfg.Model = m
	}
	return cfg, cfg.Validate()
}

// Validate checks the config is complete enough to build a working client.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("model base URL is empty (set ASTER_MODEL_BASE_URL for custom/intranet providers)")
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("model name is empty (set ASTER_MODEL_NAME)")
	}
	return nil
}

// Configured reports whether a usable model config is present.
func (c *Config) Configured() bool {
	return c != nil && c.Validate() == nil
}

// BuildClient constructs an ai.ChatClient from the resolved config.
func (c *Config) BuildClient() ai.ChatClient {
	if c.Protocol == ProtocolAnthropic {
		opts := []anthropic.Option{
			anthropic.WithURL(c.BaseURL),
			anthropic.WithURLAutoComplete(true),
			anthropic.WithAPIKey(c.APIKey),
			anthropic.WithModel(c.Model),
		}
		if c.Proxy != "" {
			opts = append(opts, anthropic.WithProxy(c.Proxy))
		}
		return anthropic.NewClient(opts...)
	}

	opts := []openai.Option{
		openai.WithURL(c.BaseURL),
		openai.WithURLAutoComplete(true),
		openai.WithAPIKey(c.APIKey),
		openai.WithModel(c.Model),
		openai.WithStream(true),
	}
	if c.Proxy != "" {
		opts = append(opts, openai.WithProxy(c.Proxy))
	}
	if c.InsecureSkipVerify {
		opts = append(opts, openai.WithInsecureSkipVerify(true))
	}
	return openai.NewClient(opts...)
}

func boolEnv(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}
