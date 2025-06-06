package logging

import (
	"encoding/json"
	"fmt"

	"github.com/canonical/pebble/client"
	"github.com/gruyaume/goops"
	"gopkg.in/yaml.v3"
)

type Integration struct {
	PebbleClient  *client.Client
	RelationName  string
	ContainerName string
}

func (i *Integration) GetRelationID() (string, error) {
	relationIDs, err := goops.GetRelationIDs(i.RelationName)
	if err != nil {
		return "", fmt.Errorf("could not get relation IDs: %w", err)
	}

	if len(relationIDs) == 0 {
		return "", fmt.Errorf("no relation IDs found for %s", i.RelationName)
	}

	return relationIDs[0], nil
}

type ProviderEndpoint struct {
	Url string `json:"url"`
}

func (i *Integration) GetEndpoint() (string, error) {
	relationID, err := i.GetRelationID()
	if err != nil {
		return "", err
	}

	relations, err := goops.ListRelations(relationID)
	if err != nil {
		return "", err
	}

	if len(relations) == 0 {
		return "", fmt.Errorf("no relations found for ID: %s", relationID)
	}

	relationData, err := goops.GetUnitRelationData(relationID, relations[0])
	if err != nil {
		goops.LogDebugf("Could not get relation data: %v", err.Error())
		return "", err
	}

	endpointStr := relationData["endpoint"]
	if endpointStr == "" {
		return "", fmt.Errorf("no endpoint found in relation data")
	}

	var providerEndpoint *ProviderEndpoint

	err = json.Unmarshal([]byte(endpointStr), &providerEndpoint)
	if err != nil {
		return "", err
	}

	if providerEndpoint == nil {
		return "", fmt.Errorf("provider endpoint is nil")
	}

	if providerEndpoint.Url == "" {
		return "", fmt.Errorf("provider endpoint URL is empty")
	}

	return providerEndpoint.Url, nil
}

type LogTarget struct {
	Override string            `yaml:"override"`
	Services []string          `yaml:"services"`
	Type     string            `yaml:"type"`
	Location string            `yaml:"location"`
	Labels   map[string]string `yaml:"labels"`
}

type PebbleLayer struct {
	LogTargets map[string]LogTarget `yaml:"log-targets"`
}

func (i *Integration) getLabels() (map[string]string, error) {
	env := goops.ReadEnv()

	meta, err := goops.ReadMetadata()
	if err != nil {
		return nil, fmt.Errorf("could not read metadata: %w", err)
	}

	labels := map[string]string{
		"product":         "Juju",
		"charm":           meta.Name,
		"juju_model":      env.ModelName,
		"juju_model_uuid": env.ModelUUID,
		"juju_unit":       env.UnitName,
	}

	return labels, nil
}

func (i *Integration) EnableEndpoints() error {
	lokiEndpoint, err := i.GetEndpoint()
	if err != nil {
		goops.LogDebugf("Could not get endpoint: %s", err.Error())
		return err
	}

	labels, err := i.getLabels()
	if err != nil {
		goops.LogDebugf("Could not get labels: %s", err.Error())
		return err
	}

	env := goops.ReadEnv()

	layerData, err := yaml.Marshal(PebbleLayer{
		LogTargets: map[string]LogTarget{
			env.UnitName: {
				Override: "replace",
				Services: []string{"all"},
				Type:     "loki",
				Location: lokiEndpoint,
				Labels:   labels,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not marshal layer data to YAML: %w", err)
	}

	err = i.PebbleClient.AddLayer(&client.AddLayerOptions{
		Combine:   true,
		Label:     i.ContainerName + "-log-forwarding",
		LayerData: layerData,
	})
	if err != nil {
		return fmt.Errorf("could not add pebble layer: %w", err)
	}

	goops.LogDebugf("Pebble layer added successfully")
	return nil
}
