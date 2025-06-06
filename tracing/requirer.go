package tracing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gruyaume/goops"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Integration struct {
	RelationName string
	ServiceName  string
}

type Protocol string

const (
	GRPC Protocol = "otlp_grpc"
	HTTP Protocol = "otlp_http"
)

type ReceiverProtocol struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ProviderReceivers struct {
	Protocol ReceiverProtocol `json:"protocol"`
	Url      string           `json:"url"`
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

func (i *Integration) PublishSupportedProtocols(protocols []Protocol) {
	relationID, err := i.GetRelationID()
	if err != nil {
		goops.LogDebugf("Could not get relation ID: %s", err.Error())
		return
	}

	var supportedProtocols []string
	for _, protocol := range protocols {
		supportedProtocols = append(supportedProtocols, string(protocol))
	}

	receiversBytes, err := json.Marshal(supportedProtocols)
	if err != nil {
		goops.LogErrorf("Could not marshal supported protocols to JSON: %s", err.Error())
		return
	}

	relationData := map[string]string{
		"receivers": string(receiversBytes),
	}

	err = goops.SetAppRelationData(relationID, relationData)
	if err != nil {
		goops.LogErrorf("Could not set relation data: %s", err.Error())
		return
	}
}

func (i *Integration) InitTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	endpoint := i.GetEndpoint()

	if endpoint == "" {
		return nil, fmt.Errorf("no gRPC receiver found")
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	sampler := sdktrace.AlwaysSample()

	env := goops.ReadEnv()

	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			semconv.ServiceNameKey.String(i.ServiceName),
			attribute.String("juju_unit", env.UnitName),
			attribute.String("juju_model", env.ModelName),
			attribute.String("juju_model_uuid", env.ModelUUID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

func (i *Integration) GetEndpoint() string {
	relationID, err := i.GetRelationID()
	if err != nil {
		goops.LogDebugf("Could not get relation ID: %s", err.Error())
		return ""
	}

	relations, err := goops.ListRelations(relationID)
	if err != nil {
		goops.LogDebugf("Could not get relation list: %s", err.Error())
		return ""
	}

	if len(relations) == 0 {
		goops.LogDebugf("No relations found for ID: %s", relationID)
		return ""
	}

	relationData, err := goops.GetAppRelationData(relationID, relations[0])
	if err != nil {
		goops.LogDebugf("Could not get relation data: %s", err.Error())
		return ""
	}

	receiversStr := relationData["receivers"]
	if receiversStr == "" {
		goops.LogDebugf("No receivers found in relation data for ID: %s", relationID)
		return ""
	}

	var providerReceivers []*ProviderReceivers

	err = json.Unmarshal([]byte(receiversStr), &providerReceivers)
	if err != nil {
		goops.LogDebugf("Could not unmarshal receivers: %s", err.Error())
		return ""
	}

	if len(providerReceivers) == 0 {
		goops.LogDebugf("No provider receivers found in relation data for ID: %s", relationID)
		return ""
	}

	for _, receiver := range providerReceivers {
		if receiver.Protocol.Name == string(GRPC) {
			return receiver.Url
		}
	}

	goops.LogDebugf("No gRPC receiver found in relation data for ID: %s", relationID)
	return ""
}
