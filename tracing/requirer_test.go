package tracing_test

import (
	"encoding/json"
	"fmt"
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

func GetEndpointExampleUse() error {
	integration := &tracing.Integration{
		RelationName: "tracing",
		ServiceName:  "my-service",
	}

	endpoint := integration.GetEndpoint()

	if endpoint != "https://tracing.example.com" {
		return fmt.Errorf("expected endpoint 'https://tracing.example.com', got '%s'", endpoint)
	}

	return nil
}

func TestPublishSupportedProtocols(t *testing.T) {
	ctx := goopstest.Context{
		Charm: PublishSupportedProtocolsExampleUse,
	}

	tracingRelation := goopstest.Relation{
		Endpoint:     "tracing",
		LocalAppData: goopstest.DataBag{},
	}

	stateIn := goopstest.State{
		Leader: true,
		Relations: []goopstest.Relation{
			tracingRelation,
		},
	}

	stateOut, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("failed to run charm: %v", err)
	}

	if ctx.CharmErr != nil {
		t.Fatalf("charm error: %v", ctx.CharmErr)
	}

	if len(stateOut.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(stateOut.Relations))
	}

	relationData, ok := stateOut.Relations[0].LocalAppData["receivers"]
	if !ok {
		t.Fatal("expected 'receivers' in relation data, but it was not found")
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

func TestGetEndpoint(t *testing.T) {
	ctx := goopstest.Context{
		Charm:   GetEndpointExampleUse,
		AppName: "requirer",
		UnitID:  "requirer/0",
	}

	tracingRelation := goopstest.Relation{
		Endpoint: "tracing",
		RemoteAppData: goopstest.DataBag{
			"receivers": `[{"url": "https://tracing.example.com", "protocol": {"name": "otlp_grpc", "type": "receiver"}}]`,
		},
	}

	stateIn := goopstest.State{
		Relations: []goopstest.Relation{
			tracingRelation,
		},
	}

	_, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("failed to run charm: %v", err)
	}
}
