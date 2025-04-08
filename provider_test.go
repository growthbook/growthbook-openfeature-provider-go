package growthbook

import (
	"context"
	"testing"
	"time"

	gb "github.com/growthbook/growthbook-golang"
	"github.com/open-feature/go-sdk/openfeature"
)

func setupTestProvider() *Provider {
	// Create a test client with JSON features
	featuresJSON := `{
		"features": {
			"bool-flag": {
				"defaultValue": true
			},
			"string-flag": {
				"defaultValue": "default-string"
			},
			"number-flag": {
				"defaultValue": 42.5
			},
			"int-flag": {
				"defaultValue": 42
			},
			"object-flag": {
				"defaultValue": {
					"key": "value"
				}
			}
		}
	}`

	gbClient, _ := gb.NewClient(
		context.Background(),
		gb.WithAttributes(gb.Attributes{
			"id": "test-user",
		}),
		gb.WithJsonFeatures(featuresJSON),
	)

	return NewProvider(gbClient, 5*time.Second)
}

func TestProviderInit(t *testing.T) {
	provider := setupTestProvider()

	// Test init
	evalCtx := openfeature.NewEvaluationContext("test-user", map[string]interface{}{
		"email": "test@example.com",
	})

	err := provider.Init(evalCtx)
	if err != nil {
		t.Fatalf("Provider initialization failed: %v", err)
	}

	if provider.Status() != openfeature.ReadyState {
		t.Errorf("Expected provider status to be ready, got %v", provider.Status())
	}
}

func TestBooleanEvaluation(t *testing.T) {
	provider := setupTestProvider()

	// Initialize the provider
	evalCtx := openfeature.NewEvaluationContext("test-user", map[string]interface{}{
		"email": "test@example.com",
	})

	_ = provider.Init(evalCtx)

	// Test boolean flag evaluation
	ctx := context.Background()
	flattenedCtx := make(map[string]interface{})
	for k, v := range evalCtx.Attributes() {
		flattenedCtx[k] = v
	}

	result := provider.BooleanEvaluation(ctx, "bool-flag", false, flattenedCtx)

	if !result.Value {
		t.Errorf("Expected boolean flag to be true, got %v", result.Value)
	}

	// Test non-existent flag
	result = provider.BooleanEvaluation(ctx, "non-existent-flag", false, flattenedCtx)

	if result.Value != false {
		t.Errorf("Expected default value for non-existent flag, got %v", result.Value)
	}

	if result.ResolutionError.Error() == "" {
		t.Error("Expected resolution error for non-existent flag")
	}
}

func TestStringEvaluation(t *testing.T) {
	provider := setupTestProvider()

	// Initialize the provider
	evalCtx := openfeature.NewEvaluationContext("test-user", map[string]interface{}{
		"email": "test@example.com",
	})

	_ = provider.Init(evalCtx)

	// Test string flag evaluation
	ctx := context.Background()
	flattenedCtx := make(map[string]interface{})
	for k, v := range evalCtx.Attributes() {
		flattenedCtx[k] = v
	}

	result := provider.StringEvaluation(ctx, "string-flag", "fallback", flattenedCtx)

	if result.Value != "default-string" {
		t.Errorf("Expected string flag to be 'default-string', got %v", result.Value)
	}
}

func TestMetadata(t *testing.T) {
	provider := setupTestProvider()

	metadata := provider.Metadata()

	if metadata.Name != "GrowthBook Provider" {
		t.Errorf("Expected provider name 'GrowthBook Provider', got %s", metadata.Name)
	}
}

func TestShutdown(t *testing.T) {
	provider := setupTestProvider()

	// Initialize
	evalCtx := openfeature.NewEvaluationContext("test-user", nil)
	_ = provider.Init(evalCtx)

	// Shutdown
	provider.Shutdown()

	if provider.Status() != openfeature.NotReadyState {
		t.Errorf("Expected provider status to be not ready after shutdown, got %v", provider.Status())
	}
}
