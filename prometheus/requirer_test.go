package prometheus_test

import (
	"fmt"
	"testing"

	"github.com/gruyaume/charm-libraries/prometheus"
	"github.com/gruyaume/goops/goopstest"
)

func WriteExampleUse() error {
	integration := &prometheus.Integration{
		RelationName: "metrics",
		CharmName:    "my-charm",
		Jobs: []*prometheus.Job{
			{
				Scheme:      "https",
				TLSConfig:   prometheus.TLSConfig{InsecureSkipVerify: true},
				MetricsPath: "/metrics",
				StaticConfigs: []prometheus.StaticConfig{
					{
						Targets: []string{"localhost:8080"},
					},
				},
			},
		},
	}

	err := integration.Write()
	if err != nil {
		return err
	}

	return nil
}

func GetScrapeMetadataExampleUse() error {
	integration := &prometheus.Integration{
		RelationName: "metrics",
		CharmName:    "my-charm",
	}

	scrapeMetadata, err := integration.GetScrapeMetadata()
	if err != nil {
		return err
	}

	expectedMetadata := &prometheus.ScrapeMetadata{
		Model:       "test-model",
		ModelUUID:   "12345",
		Application: "my-charm",
		Unit:        "my-charm/0",
		CharmName:   "my-charm",
	}

	if scrapeMetadata.Model != expectedMetadata.Model ||
		scrapeMetadata.ModelUUID != expectedMetadata.ModelUUID ||
		scrapeMetadata.Application != expectedMetadata.Application ||
		scrapeMetadata.Unit != expectedMetadata.Unit ||
		scrapeMetadata.CharmName != expectedMetadata.CharmName {
		return fmt.Errorf("scrape metadata does not match expected values")
	}

	return nil
}

func TestWriteExampleUse(t *testing.T) {
	ctx := goopstest.Context{
		Charm:   WriteExampleUse,
		AppName: "my-charm",
		UnitID:  "my-charm/0",
	}

	prometheusRelation := goopstest.Relation{
		Endpoint:     "metrics",
		LocalAppData: goopstest.DataBag{},
	}

	stateIn := goopstest.State{
		Leader: true,
		Relations: []goopstest.Relation{
			prometheusRelation,
		},
		Model: goopstest.Model{
			Name: "test-model",
			UUID: "12345",
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

	if stateOut.Relations[0].Endpoint != "metrics" {
		t.Fatalf("expected relation endpoint 'metrics', got '%s'", stateOut.Relations[0].Endpoint)
	}

	if len(stateOut.Relations[0].LocalAppData) != 2 {
		t.Fatalf("expected 2 local app data, got %d", len(stateOut.Relations[0].LocalAppData))
	}

	if scrapeJobs, ok := stateOut.Relations[0].LocalAppData["scrape_jobs"]; !ok || scrapeJobs != `[{"scheme":"https","tls_config":{"insecure_skip_verify":true},"metrics_path":"/metrics","static_configs":[{"targets":["localhost:8080"]}]}]` {
		t.Fatalf("expected scrape_jobs to be set, got %s", scrapeJobs)
	}

	if scrapeMetadata, ok := stateOut.Relations[0].LocalAppData["scrape_metadata"]; !ok || scrapeMetadata != `{"model":"test-model","model_uuid":"12345","application":"my-charm","unit":"my-charm/0","charm_name":"my-charm"}` {
		t.Fatalf("expected scrape_metadata to be set, got %s", scrapeMetadata)
	}
}

func TestGetScrapeMetadataExampleUse(t *testing.T) {
	ctx := goopstest.Context{
		Charm:   GetScrapeMetadataExampleUse,
		AppName: "my-charm",
		UnitID:  "my-charm/0",
	}

	prometheusRelation := goopstest.Relation{
		Endpoint:     "metrics",
		LocalAppData: goopstest.DataBag{},
	}

	stateIn := goopstest.State{
		Relations: []goopstest.Relation{
			prometheusRelation,
		},
		Model: goopstest.Model{
			Name: "test-model",
			UUID: "12345",
		},
	}

	stateOut, err := ctx.Run("start", stateIn)
	if err != nil {
		t.Fatalf("failed to run charm: %v", err)
	}

	if len(stateOut.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(stateOut.Relations))
	}
}
