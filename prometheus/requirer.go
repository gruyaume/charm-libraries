package prometheus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gruyaume/goops"
)

type TLSConfig struct {
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
}

type StaticConfig struct {
	Targets []string `json:"targets"`
}

type Job struct {
	Scheme        string         `json:"scheme"`
	TLSConfig     TLSConfig      `json:"tls_config"`
	MetricsPath   string         `json:"metrics_path"`
	StaticConfigs []StaticConfig `json:"static_configs"`
}

type ScrapeMetadata struct {
	Model       string `json:"model"`
	ModelUUID   string `json:"model_uuid"`
	Application string `json:"application"`
	Unit        string `json:"unit"`
	CharmName   string `json:"charm_name"`
}

type Integration struct {
	RelationName string
	Jobs         []*Job
	CharmName    string
}

func (i *Integration) GetScrapeMetadata() (*ScrapeMetadata, error) {
	env := goops.ReadEnv()

	return &ScrapeMetadata{
		Model:       env.ModelName,
		ModelUUID:   env.ModelUUID,
		Application: strings.Split(env.UnitName, "/")[0],
		Unit:        env.UnitName,
		CharmName:   i.CharmName,
	}, nil
}

func (i *Integration) Write() error {
	relationIDs, err := goops.GetRelationIDs(i.RelationName)
	if err != nil {
		return fmt.Errorf("could not get relation IDs: %w", err)
	}

	if len(relationIDs) == 0 {
		return fmt.Errorf("no relation IDs found for %s", i.RelationName)
	}

	scrapeJobs, err := json.Marshal(i.Jobs)
	if err != nil {
		return fmt.Errorf("could not marshal scrape jobs to JSON: %w", err)
	}

	scrapeMetadata, err := i.GetScrapeMetadata()
	if err != nil {
		return fmt.Errorf("could not get scrape metadata: %w", err)
	}

	scrapeMetadataBytes, err := json.Marshal(scrapeMetadata)
	if err != nil {
		return fmt.Errorf("could not marshal scrape metadata to JSON: %w", err)
	}

	relationData := map[string]string{
		"scrape_jobs":     string(scrapeJobs),
		"scrape_metadata": string(scrapeMetadataBytes),
	}

	err = goops.SetAppRelationData(relationIDs[0], relationData)
	if err != nil {
		return fmt.Errorf("could not set relation data: %w", err)
	}

	return nil
}
