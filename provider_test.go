package growthbook

import (
	"context"
	"fmt"
	"testing"
	"time"

	gb "github.com/growthbook/growthbook-golang"
	"github.com/open-feature/go-sdk/openfeature"
)

func setupTestProvider() *Provider {
	// Create a test client with JSON features in the correct format
	featuresJSON := `{
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
		},
		"rules-test": {
			"defaultValue": false,
			"rules": [
				{
					"id": "rule_id",
					"condition": {
						"email": "user@growthbook.com"
					},
					"force": true
				}
			]
		}
	}`

	gbClient, _ := gb.NewClient(
		context.Background(),
		gb.WithAttributes(gb.Attributes{
			"id": "test-user",
		}),
		gb.WithJsonFeatures(featuresJSON),
	)

	// Create provider with a short timeout and specifying false for usesDataSource
	// Since we're using in-memory features, we don't need to wait for data source loading
	provider := NewProvider(gbClient, 5*time.Second, false)

	return provider
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

// TestEvaluateFlag tests the evaluateFlag method directly
func TestEvaluateFlag(t *testing.T) {
	// Create provider with test features
	provider := setupTestProvider()

	// Initialize the provider
	evalCtx := openfeature.NewEvaluationContext("test-user", map[string]interface{}{
		"email": "test@example.com",
	})

	_ = provider.Init(evalCtx)

	// Create a flattened context
	flattenedCtx := make(map[string]interface{})
	for k, v := range evalCtx.Attributes() {
		flattenedCtx[k] = v
	}

	// Test the GrowthBook client directly
	gbClient := provider.GetClient()
	directResult := gbClient.EvalFeature(context.Background(), "bool-flag")
	fmt.Printf("DEBUG: Direct GrowthBook evaluation for bool-flag: %+v\n", directResult)

	// Test our evaluateFlag method
	feature := provider.evaluateFlag(context.Background(), "bool-flag", flattenedCtx)
	fmt.Printf("DEBUG: evaluateFlag result for bool-flag: %+v\n", feature)

	if feature == nil {
		t.Error("evaluateFlag returned nil for bool-flag")
	} else if feature.Value == nil {
		t.Error("evaluateFlag returned nil value for bool-flag")
	} else if value, ok := feature.Value.(bool); !ok || !value {
		t.Errorf("evaluateFlag returned unexpected value for bool-flag: %v (type: %T)", feature.Value, feature.Value)
	}
}

func TestEvaluateFlagWithRule(t *testing.T) {
	tests := []struct {
		name              string
		evaluationContext openfeature.FlattenedContext
		expectedResult    bool
	}{
		{
			name:              "no evaluation context",
			evaluationContext: nil,
			expectedResult:    false,
		},
		{
			name:              "matching email",
			evaluationContext: openfeature.FlattenedContext{"email": "user@growthbook.com"},
			expectedResult:    true,
		},
		{
			name:              "non matching email",
			evaluationContext: openfeature.FlattenedContext{"email": "foo@bar.com"},
			expectedResult:    false,
		},
	}

	provider := setupTestProvider()
	_ = provider.Init(openfeature.NewEvaluationContext("test-user", nil))

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			directResult := provider.evaluateFlag(context.Background(), "rules-test", tt.evaluationContext)

			if tt.expectedResult != directResult.On {
				t.Errorf("evaluateFlag returned %v, expected %v", directResult.On, tt.expectedResult)
			}
		})
	}
}

func TestEvaluateNonExistingFlag(t *testing.T) {
	provider := setupTestProvider()

	_ = provider.Init(openfeature.NewEvaluationContext("test-user", nil))
	result := provider.BooleanEvaluation(context.Background(), "non-existent-flag", false, nil)

	if result.Reason != openfeature.ErrorReason {
		t.Error("expected error reason for non-existent flag")
	}
	if result.ResolutionError.Error() != "FLAG_NOT_FOUND: flag 'non-existent-flag' not found" {
		t.Error("expected resolution error to be equal to FLAG_NOT_FOUND: flag 'non-existent-flag' not found")
	}
}
