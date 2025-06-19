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

func EnableEndpointsExampleUse() error {
	integration := &logging.Integration{
		RelationName:  "logging",
		ContainerName: "my-container",
	}

	err := integration.EnableEndpoints()
	if err != nil {
		return fmt.Errorf("failed to enable endpoints: %w", err)
	}

	return nil
}

func TestAddPebbleLayer(t *testing.T) {
	ctx := goopstest.Context{
		Charm:   EnableEndpointsExampleUse,
		AppName: "my-app",
		UnitID:  0,
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
		Containers: []*goopstest.Container{
			{
				Name:       "my-container",
				CanConnect: true,
			},
		},
		Model: &goopstest.Model{
			Name: "whatever-model",
			UUID: "whatever-model-uuid",
		},
	}

	stateOut, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("failed to run charm: %v", err)
	}

	if len(stateOut.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(stateOut.Containers))
	}

	container := stateOut.Containers[0]
	if container.Name != "my-container" {
		t.Fatalf("expected container name 'my-container', got '%s'", container.Name)
	}

	if container.Layers == nil {
		t.Fatal("expected container to have layers, got nil")
	}

	layer, exists := container.Layers["my-container-log-forwarding"]
	if !exists {
		t.Fatal("expected container to have 'my-container-log-forwarding' layer, but it does not exist")
	}

	if layer.LogTargets == nil {
		t.Fatal("expected layer to have log targets, got nil")
	}

	if len(layer.LogTargets) != 1 {
		t.Fatalf("expected 1 log target, got %d", len(layer.LogTargets))
	}

	logTarget, exists := layer.LogTargets["my-app/0"]
	if !exists {
		t.Fatal("expected layer to have 'my-app/0' log target, but it does not exist")
	}

	if logTarget.Override != "replace" {
		t.Fatalf("expected log target override to be 'replace', got '%s'", logTarget.Override)
	}

	if logTarget.Type != "loki" {
		t.Fatalf("expected log target type to be 'loki', got '%s'", logTarget.Type)
	}

	if logTarget.Location != "https://logging.example.com" {
		t.Fatalf("expected log target location to be 'https://logging.example.com', got '%s'", logTarget.Location)
	}

	if len(logTarget.Services) != 1 || logTarget.Services[0] != "all" {
		t.Fatalf("expected log target services to contain 'all', got %v", logTarget.Services)
	}

	if logTarget.Labels == nil {
		t.Fatal("expected log target to have labels, got nil")
	}

	if logTarget.Labels["juju_model"] != "whatever-model" {
		t.Fatalf("expected log target label 'juju_model' to be 'whatever-model', got '%s'", logTarget.Labels["juju_model"])
	}

	if logTarget.Labels["juju_unit"] != "my-app/0" {
		t.Fatalf("expected log target label 'juju_unit' to be 'my-app/0', got '%s'", logTarget.Labels["juju_unit"])
	}

	if logTarget.Labels["product"] != "Juju" {
		t.Fatalf("expected log target label 'product' to be 'Juju', got '%s'", logTarget.Labels["product"])
	}

	if logTarget.Labels["juju_model_uuid"] != "whatever-model-uuid" {
		t.Fatalf("expected log target label 'juju_model_uuid' to be 'whatever-model-uuid', got '%s'", logTarget.Labels["juju_model_uuid"])
	}

}
