package tracing_test

import (
	"encoding/json"
	"testing"

	"github.com/gruyaume/charm-libraries/tracing"
	"github.com/gruyaume/goops/goopstest"
)

func PublishSupportedProtocolsExampleUse() error {
	integration := &tracing.Integration{
		RelationName: "tracing",
		ServiceName:  "my-service",
	}

	protocols := []tracing.Protocol{tracing.GRPC, tracing.HTTP}
	integration.PublishSupportedProtocols(protocols)

	return nil
}

func TestPublishSupportedProtocols(t *testing.T) {
	ctx := goopstest.Context{
		Charm: PublishSupportedProtocolsExampleUse,
	}

	tracingRelation := &goopstest.Relation{
		Endpoint:     "tracing",
		LocalAppData: goopstest.DataBag{},
	}

	stateIn := &goopstest.State{
		Relations: []*goopstest.Relation{
			tracingRelation,
		},
	}

	stateOut, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("failed to run charm: %v", err)
	}

	if len(stateOut.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(stateOut.Relations))
	}

	relationData, ok := stateOut.Relations[0].LocalAppData["receivers"]
	if !ok {
		t.Fatal("expected 'provider' key in relation data")
	}

	var supportedProtocols []string

	err = json.Unmarshal([]byte(relationData), &supportedProtocols)
	if err != nil {
		t.Fatalf("failed to unmarshal relation data: %v", err)
	}

	if len(supportedProtocols) != 2 {
		t.Fatalf("expected 2 supported protocols, got %d", len(supportedProtocols))
	}

	if supportedProtocols[0] != string(tracing.GRPC) && supportedProtocols[0] != string(tracing.HTTP) {
		t.Fatalf("expected one of the protocols to be 'otlp_grpc' or 'otlp_http', got '%s'", supportedProtocols[0])
	}

	if supportedProtocols[1] != string(tracing.GRPC) && supportedProtocols[1] != string(tracing.HTTP) {
		t.Fatalf("expected one of the protocols to be 'otlp_grpc' or 'otlp_http', got '%s'", supportedProtocols[1])
	}

}
