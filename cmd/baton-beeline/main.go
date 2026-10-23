package main

import (
	"context"

	cfg "github.com/conductorone/baton-beeline/pkg/config"
	"github.com/conductorone/baton-beeline/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

var version = "dev"

func main() {
	ctx := context.Background()
	config.RunConnector(
		ctx,
		"baton-beeline",
		version,
		cfg.Config,
		getConnector,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.Connector{}),
	)
}

func getConnector(ctx context.Context, bc *cfg.Beeline, _ *cli.ConnectorOpts) (connectorbuilder.ConnectorBuilderV2, []connectorbuilder.Opt, error) {
	l := ctxzap.Extract(ctx)
	if err := field.Validate(cfg.Config, bc); err != nil {
		return nil, nil, err
	}

	cb, err := connector.New(
		ctx,
		bc.BaseUrl,
		bc.AuthServerUrl,
		bc.BeelineClientId,
		bc.BeelineClientSecret,
		bc.BeelineClientSiteId,
	)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, nil, err
	}

	return cb, nil, nil
}
