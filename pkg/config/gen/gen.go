package main

import (
	cfg "github.com/conductorone/baton-beeline/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("beeline", cfg.Config)
}
