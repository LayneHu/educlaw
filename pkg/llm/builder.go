package llm

import (
	"log"
	"strings"

	"github.com/pingjie/educlaw/pkg/config"
)

// BuildProviderFromConfig constructs a Provider from the model_list configuration.
// Returns a DisabledProvider if no valid API key is configured.
func BuildProviderFromConfig(cfg *config.Config) Provider {
	var providers []Provider

	if len(cfg.ModelList) > 0 {
		primaryCfg, modelName, err := cfg.ResolveModelSelection()
		if err == nil {
			if primaryCfg.APIKey != "" || strings.EqualFold(primaryCfg.Provider, "ollama") {
				providers = append(providers, NewClient(primaryCfg))
			}
			fallbacks, fbErr := cfg.ResolveFallbackSelections()
			if fbErr != nil {
				log.Printf("Warning: resolving fallback models failed: %v", fbErr)
			} else {
				for _, fb := range fallbacks {
					if fb.APIKey != "" || strings.EqualFold(fb.Provider, "ollama") {
						modelCfg := fb
						providers = append(providers, NewClient(&modelCfg))
					}
				}
			}
			log.Printf("LLM: primary=%s -> %s, fallbacks=%d", modelName, primaryCfg.Model, max(0, len(providers)-1))
		} else {
			log.Printf("Warning: resolving model_list failed: %v", err)
		}
	}

	if len(providers) == 0 {
		log.Printf("Warning: no LLM API key configured — AI responses will fail")
		return NewDisabledProvider("missing model_list api key")
	}
	if len(providers) == 1 {
		return providers[0]
	}
	return NewMultiProvider(providers...)
}
