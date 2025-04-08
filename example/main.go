package main

import (
	"context"
	"fmt"
	"log"
	"time"

	gb "github.com/growthbook/growthbook-golang"
	gbprovider "github.com/growthbook/growthbook-openfeature-provider-go"
	"github.com/open-feature/go-sdk/openfeature"
)

func main() {
	// Create a context with timeout for the initialization phase
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()

	log.Println("Creating GrowthBook client...")
	// Create a GrowthBook client with built-in data source
	gbClient, err := gb.NewClient(
		initCtx,                        // Use the initialization context
		gb.WithClientKey("sdk-abc123"), // Replace with your actual client key
		// Optional: API host (defaults to cdn.growthbook.io)
		gb.WithApiHost("https://cdn.growthbook.io"),
		// Choose one of these data sources:
		gb.WithSseDataSource(), // Server-Sent Events (SSE) for real-time updates
		// gb.WithPollDataSource(30*time.Second), // Or polling with specified interval
		gb.WithAttributes(map[string]interface{}{ // Set default attributes
			"id":    "user-123",
			"email": "user@example.com",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create GrowthBook client: %v", err)
	}
	// Close the client when done
	defer gbClient.Close()

	// The data source starts asynchronously. Wait until data is loaded
	// Note: we're using the initialization context here
	log.Println("Waiting for GrowthBook client to load features...")
	if err := gbClient.EnsureLoaded(initCtx); err != nil {
		log.Fatalf("Data loading failed: %v", err)
	}
	log.Println("GrowthBook client features loaded successfully")

	// Create our GrowthBook provider with a 20-second timeout for initialization
	// The second parameter (true) explicitly indicates that this client uses a data source
	provider := gbprovider.NewProvider(gbClient, 20*time.Second, true)

	// Register with OpenFeature
	log.Println("Registering GrowthBook provider with OpenFeature...")
	err = openfeature.SetProvider(provider)
	if err != nil {
		log.Fatalf("Failed to set provider: %v", err)
	}

	// Create a proper evaluation context for initialization
	attributes := map[string]interface{}{
		"id":    "user-123",
		"email": "user@example.com",
	}
	evalContext := openfeature.NewEvaluationContext("user-123", attributes)

	// Initialize the provider
	// The provider uses its own context with timeout for feature loading
	log.Println("Initializing the provider...")
	if err := provider.Init(evalContext); err != nil {
		// Properly handle initialization errors
		if providerErr, ok := err.(*openfeature.ProviderInitError); ok {
			log.Fatalf("Provider initialization failed with code %s: %s",
				providerErr.ErrorCode, providerErr.Message)
		} else {
			log.Fatalf("Provider initialization failed: %v", err)
		}
	}

	// Check provider status
	providerStatus := provider.Status()
	log.Printf("Provider status: %v", providerStatus)

	if providerStatus != openfeature.ReadyState {
		log.Fatalf("Provider not ready. Current state: %v", providerStatus)
	}

	// Create an OpenFeature client
	client := openfeature.NewClient("example-app")

	// Create an evaluation context for flag evaluation
	evalCtx := openfeature.NewEvaluationContext("user-123", map[string]interface{}{
		"email": "user@example.com",
	})

	// Create a new context for flag evaluation with a shorter timeout
	evalTimeoutCtx, evalCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer evalCancel()

	// Evaluate flags with proper error handling and context
	// Evaluate boolean flag
	boolResult, err := client.BooleanValueDetails(evalTimeoutCtx, "show-welcome-banner", false, evalCtx)
	if err != nil {
		log.Printf("Error evaluating boolean flag: %v", err)
	} else {
		fmt.Printf("Boolean flag 'show-welcome-banner' value: %v (reason: %s)\n",
			boolResult.Value, boolResult.Reason)

		// Print metadata if available
		if len(boolResult.FlagMetadata) > 0 {
			fmt.Printf("  - Metadata: %v\n", boolResult.FlagMetadata)
		}
	}

	// Evaluate string flag with a flag that might not exist to show error handling
	stringResult, err := client.StringValueDetails(evalTimeoutCtx, "non-existent-flag", "default-value", evalCtx)
	if err != nil {
		log.Printf("Error evaluating string flag: %v", err)
	} else {
		fmt.Printf("String flag 'non-existent-flag' value: %s (reason: %s)\n",
			stringResult.Value, stringResult.Reason)
	}

	// Evaluate string flag that should exist
	stringResult, err = client.StringValueDetails(evalTimeoutCtx, "header-color", "blue", evalCtx)
	if err != nil {
		log.Printf("Error evaluating string flag: %v", err)
	} else {
		fmt.Printf("String flag 'header-color' value: %s (reason: %s)\n",
			stringResult.Value, stringResult.Reason)
	}

	// Evaluate integer flag
	intResult, err := client.IntValueDetails(evalTimeoutCtx, "results-per-page", 10, evalCtx)
	if err != nil {
		log.Printf("Error evaluating int flag: %v", err)
	} else {
		fmt.Printf("Int flag 'results-per-page' value: %d (reason: %s)\n",
			intResult.Value, intResult.Reason)
	}

	// Demonstrate shutdown
	log.Println("Shutting down provider...")
	provider.Shutdown()
	log.Printf("Provider shut down. Status: %v", provider.Status())
}
