package logging_test

import (
	"fmt"
	"testing"

	"github.com/gruyaume/charm-libraries/logging"
	"github.com/gruyaume/goops/goopstest"
)

func GetEndpointExampleUse() error {
	integration := &logging.Integration{
		RelationName: "logging",
	}

	endpoint, err := integration.GetEndpoint()
	if err != nil {
		return fmt.Errorf("failed to get endpoint: %w", err)
	}

	if endpoint != "https://logging.example.com" {
		return fmt.Errorf("expected endpoint 'https://logging.example.com', got '%s'", endpoint)
	}

	return nil
}

func TestGetEndpoint(t *testing.T) {
	ctx := goopstest.Context{
		Charm: GetEndpointExampleUse,
	}

	loggingRelation := &goopstest.Relation{
		Endpoint: "logging",
		RemoteUnitsData: map[goopstest.UnitID]goopstest.DataBag{
			"provider/0": {
				"endpoint": `{"url": "https://logging.example.com"}`,
			},
		},
	}

	stateIn := &goopstest.State{
		Relations: []*goopstest.Relation{
			loggingRelation,
		},
	}

	_, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("failed to run charm: %v", err)
	}
}
