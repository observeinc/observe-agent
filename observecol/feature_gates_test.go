package observecol

// This file contains unit tests for the observe-agent feature flags functionality.
//
// The tests verify that:
// 1. All feature flags defined in internalFeatureFlagDefaults exist in the OpenTelemetry
//    Collector's global feature gate registry
// 2. All feature flags can be set to their configured default values
// 3. The ApplyFeatureGates function correctly applies feature gate settings
// 4. User-specified feature gate values take precedence over defaults
// 5. Invalid feature gates are properly rejected with appropriate error messages
//
// These tests provide an automated way to verify that all feature flags we are
// setting exist and are able to be set to the values we want, preventing runtime
// errors when the agent starts up.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/featuregate"
)

// TestInternalFeatureFlagDefaultsExist verifies that all feature flags defined in
// internalFeatureFlagDefaults actually exist in the OpenTelemetry Collector's
// global feature gate registry.
//
// This test will fail if:
// - A feature flag ID is misspelled in internalFeatureFlagDefaults
// - A feature flag has been removed from the OpenTelemetry Collector
// - A feature flag has been renamed in the OpenTelemetry Collector
func TestInternalFeatureFlagDefaultsExist(t *testing.T) {
	// Collect all registered feature gates
	registeredGates := make(map[string]*featuregate.Gate)
	featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
		registeredGates[g.ID()] = g
	})

	// Verify each internal feature flag exists in the registry
	for flagID := range internalFeatureFlagDefaults {
		t.Run(flagID, func(t *testing.T) {
			gate, exists := registeredGates[flagID]
			assert.True(t, exists, "Feature flag %q is not registered in the global registry. Valid gates: %v", flagID, getRegisteredGateIDs())
			if exists {
				t.Logf("Feature gate %q found - Stage: %s, Description: %s", flagID, gate.Stage(), gate.Description())
			}
		})
	}
}

// TestInternalFeatureFlagDefaultsCanBeSet verifies that all feature flags can be
// set to their default values without errors.
//
// This test will fail if:
//   - A feature flag cannot be set to its configured default value
//   - A feature flag's lifecycle stage prevents it from being set as expected
//     (e.g., trying to disable a Stable gate or enable a Deprecated gate)
func TestInternalFeatureFlagDefaultsCanBeSet(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() { resetFeatureGates(t, ctx) })

	for flagID, defaultValue := range internalFeatureFlagDefaults {
		t.Run(flagID, func(t *testing.T) {
			// Try to set the feature gate to its default value
			err := featuregate.GlobalRegistry().Set(flagID, defaultValue)

			// The error handling depends on the stage of the feature gate
			gate, exists := getGate(flagID)
			require.True(t, exists, "Feature gate %q should exist", flagID)

			// For Alpha and Beta stages, setting should succeed
			assert.NoError(t, err, "Failed to set feature gate %q to %v", flagID, defaultValue)

			// Verify the gate is set to the expected value
			assert.Equal(t, defaultValue, gate.IsEnabled(), "Feature gate %q should be set to %v", flagID, defaultValue)

			t.Logf("Feature gate %q - Stage: %s, Default: %v, Current: %v",
				flagID, gate.Stage(), defaultValue, gate.IsEnabled())
		})
	}
}

// TestApplyFeatureGates tests the ApplyFeatureGates function with various scenarios.
//
// This test verifies:
// - Empty feature gates list applies only defaults
// - Feature gates can be enabled with '+' prefix or no prefix
// - Feature gates can be disabled with '-' prefix
// - Invalid feature gate IDs are rejected with appropriate errors
// - Multiple feature gates can be applied together
// - User-specified values override defaults
//
// This is the most comprehensive test that validates the entire feature gate
// application workflow as it would be used in production.
func TestApplyFeatureGates(t *testing.T) {
	tests := []struct {
		name          string
		featureGates  []string
		expectError   bool
		errorContains string
		checkGates    map[string]bool // gates to check and their expected values
	}{
		{
			name:         "empty feature gates",
			featureGates: []string{},
			expectError:  false,
			checkGates:   internalFeatureFlagDefaults, // Should apply defaults
		},
		{
			name:         "enable gate with + prefix",
			featureGates: []string{"+connector.spanmetrics.useSecondAsDefaultMetricsUnit"},
			expectError:  false,
			checkGates: map[string]bool{
				"connector.spanmetrics.useSecondAsDefaultMetricsUnit": true,
			},
		},
		{
			name:         "enable gate without prefix",
			featureGates: []string{"connector.spanmetrics.useSecondAsDefaultMetricsUnit"},
			expectError:  false,
			checkGates: map[string]bool{
				"connector.spanmetrics.useSecondAsDefaultMetricsUnit": true,
			},
		},
		{
			name:         "disable gate with - prefix",
			featureGates: []string{"-exporter.prometheusremotewritexporter.EnableMultipleWorkers"},
			expectError:  false,
			checkGates: map[string]bool{
				"exporter.prometheusremotewritexporter.EnableMultipleWorkers": false,
			},
		},
		{
			name:          "invalid feature gate",
			featureGates:  []string{"invalid.feature.gate.that.does.not.exist"},
			expectError:   true,
			errorContains: "no such feature gate",
		},
		{
			name: "multiple gates",
			featureGates: []string{
				"connector.spanmetrics.useSecondAsDefaultMetricsUnit",
				"-connector.spanmetrics.excludeResourceMetrics",
			},
			expectError: false,
			checkGates: map[string]bool{
				"connector.spanmetrics.useSecondAsDefaultMetricsUnit": true,
				"connector.spanmetrics.excludeResourceMetrics":        false,
			},
		},
		{
			name: "user override takes precedence over defaults",
			featureGates: []string{
				"-exporter.prometheusremotewritexporter.EnableMultipleWorkers", // default is true
			},
			expectError: false,
			checkGates: map[string]bool{
				"exporter.prometheusremotewritexporter.EnableMultipleWorkers": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Set the global featureGates variable
			featureGates = tt.featureGates
			t.Cleanup(func() { resetFeatureGates(t, ctx) })

			// Apply feature gates
			err := ApplyFeatureGates(ctx)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err, "ApplyFeatureGates should not return error for non-error test cases")
			}

			// Verify expected gate states
			for gateID, expectedValue := range tt.checkGates {
				gate, exists := getGate(gateID)
				if !exists {
					t.Errorf("Gate %q does not exist", gateID)
					continue
				}

				assert.Equal(t, expectedValue, gate.IsEnabled(),
					"Gate %q should be %v (Stage: %s)", gateID, expectedValue, gate.Stage())
			}
		})
	}
}

// Helper function to get a feature gate by ID
func getGate(id string) (*featuregate.Gate, bool) {
	var foundGate *featuregate.Gate
	var found bool

	featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
		if g.ID() == id {
			foundGate = g
			found = true
		}
	})

	return foundGate, found
}

// Helper function to get all registered gate IDs
func getRegisteredGateIDs() []string {
	var ids []string
	featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
		ids = append(ids, g.ID())
	})
	return ids
}

// Helper function to reset feature gates to their defaults after tests
func resetFeatureGates(t *testing.T, ctx context.Context) {
	t.Helper()

	// Reset the global featureGates variable
	featureGates = []string{}

	// Reapply defaults
	for id, enabled := range internalFeatureFlagDefaults {
		gate, exists := getGate(id)
		if !exists {
			continue
		}

		// Only reset if the gate can be set (not stable when trying to disable, etc.)
		if gate.Stage() == featuregate.StageStable && !enabled {
			continue // Can't disable stable gates
		}
		if gate.Stage() == featuregate.StageDeprecated && enabled {
			continue // Can't enable deprecated gates
		}

		_ = featuregate.GlobalRegistry().Set(id, enabled)
	}
}
