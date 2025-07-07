package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	baseURLField = field.StringField(
		"base-url",
		field.WithDisplayName("Base URL"),
		field.WithDescription("The Beeline base URL."),
		field.WithRequired(false),
		field.WithDefaultValue("https://client.beeline.com"),
	)
	beelineClientSiteIDField = field.StringField(
		"beeline-client-site-id",
		field.WithDisplayName("Client Site ID"),
		field.WithDescription("The Beeline client site ID."),
		field.WithRequired(true),
	)
	beelineClientIDField = field.StringField(
		"beeline-client-id",
		field.WithDisplayName("Client ID"),
		field.WithDescription("The OAuth2 client ID for Beeline API access."),
		field.WithRequired(true),
	)
	beelineClientSecretField = field.StringField(
		"beeline-client-secret",
		field.WithDisplayName("Client Secret"),
		field.WithDescription("The OAuth2 client secret for Beeline API access."),
		field.WithRequired(true),
		field.WithIsSecret(true),
	)
	authServerURLField = field.StringField(
		"auth-server-url",
		field.WithDisplayName("Auth Server URL"),
		field.WithDescription("The Beeline auth server URL."),
		field.WithRequired(false),
		field.WithDefaultValue("https://integrations.auth.beeline.com/oauth/token"),
	)
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{
		baseURLField,
		authServerURLField,
		beelineClientSiteIDField,
		beelineClientIDField,
		beelineClientSecretField,
	},
	field.WithConnectorDisplayName("Beeline"),
	field.WithHelpUrl("/docs/baton/beeline"),
	field.WithIconUrl("/static/app-icons/beeline.svg"),
)
