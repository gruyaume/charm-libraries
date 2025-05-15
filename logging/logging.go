package logging

import (
	"encoding/json"
	"fmt"

	"github.com/canonical/pebble/client"
	"github.com/gruyaume/goops"
	"github.com/gruyaume/goops/commands"
	"github.com/gruyaume/goops/metadata"
	"gopkg.in/yaml.v3"
)

type Integration struct {
	HookContext   *goops.HookContext
	PebbleClient  *client.Client
	RelationName  string
	ContainerName string
}

func (i *Integration) GetRelationID() (string, error) {
	relationIDs, err := i.HookContext.Commands.RelationIDs(&commands.RelationIDsOptions{
		Name: i.RelationName,
	})
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

	relations, err := i.HookContext.Commands.RelationList(&commands.RelationListOptions{
		ID: relationID,
	})
	if err != nil {
		return "", err
	}

	if len(relations) == 0 {
		return "", fmt.Errorf("no relations found for ID: %s", relationID)
	}

	relationData, err := i.HookContext.Commands.RelationGet(&commands.RelationGetOptions{
		ID:     relationID,
		UnitID: relations[0],
		App:    false,
	})
	if err != nil {
		i.HookContext.Commands.JujuLog(commands.Debug, "Could not get relation data:", err.Error())
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
	unitName := i.HookContext.Environment.JujuUnitName()

	metadata, err := metadata.GetCharmMetadata(i.HookContext.Environment)
	if err != nil {
		return nil, fmt.Errorf("could not get charm metadata: %w", err)
	}

	modelName := i.HookContext.Environment.JujuModelName()
	modelUUID := i.HookContext.Environment.JujuModelUUID()

	labels := map[string]string{
		"product":         "Juju",
		"charm":           metadata.Name,
		"juju_model":      modelName,
		"juju_model_uuid": modelUUID,
		"juju_unit":       unitName,
	}

	return labels, nil
}

func (i *Integration) EnableEndpoints() error {
	lokiEndpoint, err := i.GetEndpoint()
	if err != nil {
		i.HookContext.Commands.JujuLog(commands.Debug, "Could not get endpoint:", err.Error())
		return err
	}

	labels, err := i.getLabels()
	if err != nil {
		i.HookContext.Commands.JujuLog(commands.Debug, "Could not get labels:", err.Error())
		return err
	}

	unitName := i.HookContext.Environment.JujuUnitName()

	layerData, err := yaml.Marshal(PebbleLayer{
		LogTargets: map[string]LogTarget{
			unitName: {
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

	i.HookContext.Commands.JujuLog(commands.Debug, "Pebble layer added successfully")

	return nil
}
