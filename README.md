# GrowthBook OpenFeature Provider for Go

This is an OpenFeature provider implementation for [GrowthBook](https://www.growthbook.io/), a feature flagging and A/B testing platform.

## Installation

```bash
go get github.com/growthbook/growthbook-openfeature-provider-go
```

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"

    gb "github.com/growthbook/growthbook-golang"
    gbprovider "github.com/growthbook/growthbook-openfeature-provider-go"
    "github.com/open-feature/go-sdk/openfeature"
)

func main() {
    // Create a GrowthBook client
    gbClient, err := gb.NewClient(context.Background(),
        gb.WithAPIHost("https://cdn.growthbook.io"),
        gb.WithClientKey("YOUR_CLIENT_KEY"),
    )
    if err != nil {
        log.Fatalf("Failed to create GrowthBook client: %v", err)
    }

    // Create the GrowthBook provider
    provider := gbprovider.NewProvider(gbClient)

    // Register the provider with OpenFeature
    err = openfeature.SetProvider(provider)
    if err != nil {
        log.Fatalf("Failed to set provider: %v", err)
    }

    // Create an OpenFeature client
    client := openfeature.NewClient("example-app")

    // Create an evaluation context
    evalCtx := openfeature.NewEvaluationContext("user-123", map[string]interface{}{
        "email": "user@example.com",
        "country": "US",
    })

    // Evaluate flags
    boolValue, err := client.BooleanValue(context.Background(), "feature-flag-key", false, evalCtx)
    if err != nil {
        log.Printf("Error evaluating flag: %v", err)
    } else {
        fmt.Printf("Feature flag value: %v\n", boolValue)
    }
}
```

### Using In-Memory Feature Flags

You can also initialize the GrowthBook client with in-memory feature flags for testing:

```go
gbClient, err := gb.NewClient(context.Background(),
    gb.WithAttributes(map[string]interface{}{
        "id": "user-123",
    }),
    gb.WithFeatures(map[string]interface{}{
        "my-feature": map[string]interface{}{
            "defaultValue": false,
            "rules": []map[string]interface{}{
                {
                    "condition": map[string]interface{}{
                        "id": "user-123",
                    },
                    "force": true,
                },
            },
        },
    }),
)
```

### Getting Feature Value Details

To get more information about flag evaluation:

```go
valueDetails, err := client.BooleanValueDetails(context.Background(), "feature-flag-key", false, evalCtx)
if err != nil {
    log.Printf("Error evaluating flag: %v", err)
} else {
    fmt.Printf("Value: %v\n", valueDetails.Value)
    fmt.Printf("Reason: %s\n", valueDetails.Reason)
    fmt.Printf("Variant: %s\n", valueDetails.Variant)
}
```

## Features

This provider supports:

- Boolean, string, number (float/int), and object flag types
- Evaluation contexts for targeting and segmentation
- Flag metadata
- Experiment tracking
- Remote feature configurations via GrowthBook SDK

## License

MIT
