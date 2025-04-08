// Package growthbook provides an OpenFeature provider implementation for the GrowthBook feature flagging system.
package growthbook

import (
	"context"
	"fmt"
	"sync"
	"time"

	gb "github.com/growthbook/growthbook-golang"
	"github.com/open-feature/go-sdk/openfeature"
)

// Provider implements the OpenFeature provider interface for GrowthBook.
type Provider struct {
	gbClient   *gb.Client
	state      openfeature.State
	stateMutex sync.RWMutex
	timeout    time.Duration // Timeout for feature loading
}

// Metadata returns metadata about the provider.
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "GrowthBook Provider",
	}
}

// NewProvider creates a new instance of the GrowthBook OpenFeature provider.
// You can specify an optional timeout for feature loading during initialization.
func NewProvider(gbClient *gb.Client, timeout ...time.Duration) *Provider {
	// Default timeout is 30 seconds
	loadTimeout := 30 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		loadTimeout = timeout[0]
	}

	return &Provider{
		gbClient: gbClient,
		state:    openfeature.NotReadyState,
		timeout:  loadTimeout,
	}
}

// Hooks returns any hooks the provider wishes to register.
func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

// Init initializes the provider
func (p *Provider) Init(evalCtx openfeature.EvaluationContext) error {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	// Set state to not ready initially
	p.state = openfeature.NotReadyState

	// Get attributes from evaluation context
	attrs := evalCtx.Attributes()
	if len(attrs) > 0 {
		p.gbClient.WithAttributes(gb.Attributes(attrs))
	}

	// Create a context with a reasonable timeout for loading features
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	// If the client has a data source, ensure it's loaded
	if err := p.gbClient.EnsureLoaded(ctx); err != nil {
		p.state = openfeature.ErrorState
		return &openfeature.ProviderInitError{
			ErrorCode: openfeature.ProviderFatalCode,
			Message:   fmt.Sprintf("failed to load GrowthBook features: %v", err),
		}
	}

	// Mark as ready
	p.state = openfeature.ReadyState
	return nil
}

// Status returns the current provider status
func (p *Provider) Status() openfeature.State {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return p.state
}

// Shutdown cleans up any resources used by the provider
func (p *Provider) Shutdown() {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	// Close the GrowthBook client to clean up resources
	p.gbClient.Close()

	// Set state to not ready on shutdown
	p.state = openfeature.NotReadyState
}

// BooleanEvaluation evaluates a boolean feature flag.
func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	// Check if provider is ready
	if p.Status() != openfeature.ReadyState {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewProviderNotReadyResolutionError("GrowthBook provider is not ready"),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	feature := p.evaluateFlag(ctx, flag, evalCtx)

	// Flag not found
	if feature == nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewFlagNotFoundResolutionError(fmt.Sprintf("flag '%s' not found", flag)),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	if feature.Value != nil {
		if value, ok := feature.Value.(bool); ok {
			return openfeature.BoolResolutionDetail{
				Value:                    value,
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		}

		// Type mismatch
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(
					fmt.Sprintf("flag '%s' exists but is not a boolean value", flag)),
				Reason: openfeature.ErrorReason,
			},
		}
	}

	return openfeature.BoolResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: createDefaultResolutionDetail(),
	}
}

// StringEvaluation evaluates a string feature flag.
func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	// Check if provider is ready
	if p.Status() != openfeature.ReadyState {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewProviderNotReadyResolutionError("GrowthBook provider is not ready"),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	feature := p.evaluateFlag(ctx, flag, evalCtx)

	// Flag not found
	if feature == nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewFlagNotFoundResolutionError(fmt.Sprintf("flag '%s' not found", flag)),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	if feature.Value != nil {
		if value, ok := feature.Value.(string); ok {
			return openfeature.StringResolutionDetail{
				Value:                    value,
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		}

		// Type mismatch
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(
					fmt.Sprintf("flag '%s' exists but is not a string value", flag)),
				Reason: openfeature.ErrorReason,
			},
		}
	}

	return openfeature.StringResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: createDefaultResolutionDetail(),
	}
}

// FloatEvaluation evaluates a float feature flag.
func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	// Check if provider is ready
	if p.Status() != openfeature.ReadyState {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewProviderNotReadyResolutionError("GrowthBook provider is not ready"),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	feature := p.evaluateFlag(ctx, flag, evalCtx)

	// Flag not found
	if feature == nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewFlagNotFoundResolutionError(fmt.Sprintf("flag '%s' not found", flag)),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	if feature.Value != nil {
		switch v := feature.Value.(type) {
		case float64:
			return openfeature.FloatResolutionDetail{
				Value:                    v,
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		case float32:
			return openfeature.FloatResolutionDetail{
				Value:                    float64(v),
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		case int:
			return openfeature.FloatResolutionDetail{
				Value:                    float64(v),
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		default:
			// Type mismatch
			return openfeature.FloatResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					ResolutionError: openfeature.NewTypeMismatchResolutionError(
						fmt.Sprintf("flag '%s' exists but is not a numeric value", flag)),
					Reason: openfeature.ErrorReason,
				},
			}
		}
	}

	return openfeature.FloatResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: createDefaultResolutionDetail(),
	}
}

// IntEvaluation evaluates an integer feature flag.
func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	// Check if provider is ready
	if p.Status() != openfeature.ReadyState {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewProviderNotReadyResolutionError("GrowthBook provider is not ready"),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	feature := p.evaluateFlag(ctx, flag, evalCtx)

	// Flag not found
	if feature == nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewFlagNotFoundResolutionError(fmt.Sprintf("flag '%s' not found", flag)),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	if feature.Value != nil {
		switch v := feature.Value.(type) {
		case int64:
			return openfeature.IntResolutionDetail{
				Value:                    v,
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		case int:
			return openfeature.IntResolutionDetail{
				Value:                    int64(v),
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		case float64:
			return openfeature.IntResolutionDetail{
				Value:                    int64(v),
				ProviderResolutionDetail: createResolutionDetail(feature),
			}
		default:
			// Type mismatch
			return openfeature.IntResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
					ResolutionError: openfeature.NewTypeMismatchResolutionError(
						fmt.Sprintf("flag '%s' exists but is not a numeric value", flag)),
					Reason: openfeature.ErrorReason,
				},
			}
		}
	}

	return openfeature.IntResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: createDefaultResolutionDetail(),
	}
}

// ObjectEvaluation evaluates an object feature flag.
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	// Check if provider is ready
	if p.Status() != openfeature.ReadyState {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewProviderNotReadyResolutionError("GrowthBook provider is not ready"),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	feature := p.evaluateFlag(ctx, flag, evalCtx)

	// Flag not found
	if feature == nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewFlagNotFoundResolutionError(fmt.Sprintf("flag '%s' not found", flag)),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	if feature.Value != nil {
		return openfeature.InterfaceResolutionDetail{
			Value:                    feature.Value,
			ProviderResolutionDetail: createResolutionDetail(feature),
		}
	}

	return openfeature.InterfaceResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: createDefaultResolutionDetail(),
	}
}

// evaluateFlag calls GrowthBook's feature evaluation
func (p *Provider) evaluateFlag(ctx context.Context, flag string, evalCtx openfeature.FlattenedContext) *gb.FeatureResult {
	// Set attributes from evalCtx to GrowthBook
	gbContext := make(map[string]interface{})

	// Convert evalCtx to GrowthBook attributes
	for k, v := range evalCtx {
		gbContext[k] = v
	}

	// Update GrowthBook context
	p.gbClient.WithAttributes(gbContext)

	// Evaluate the feature in GrowthBook
	return p.gbClient.EvalFeature(ctx, flag)
}

// createResolutionDetail creates a ProviderResolutionDetail from a GrowthBook feature result
func createResolutionDetail(feature *gb.FeatureResult) openfeature.ProviderResolutionDetail {
	reason := openfeature.DefaultReason
	if feature.Source != "" && feature.Source != gb.UnknownFeatureResultSource && feature.Source != gb.DefaultValueResultSource {
		reason = openfeature.TargetingMatchReason
	}

	metadata := openfeature.FlagMetadata{
		"source":     string(feature.Source),
		"experiment": feature.InExperiment(),
	}

	// We'll use RuleId as the variant since GrowthBook doesn't have a direct "variation ID" concept
	variant := feature.RuleId

	return openfeature.ProviderResolutionDetail{
		Reason:       reason,
		Variant:      variant,
		FlagMetadata: metadata,
	}
}

// createDefaultResolutionDetail creates a default ProviderResolutionDetail
func createDefaultResolutionDetail() openfeature.ProviderResolutionDetail {
	return openfeature.ProviderResolutionDetail{
		Reason: openfeature.DefaultReason,
	}
}

// GetClient returns the underlying GrowthBook client
func (p *Provider) GetClient() *gb.Client {
	return p.gbClient
}
