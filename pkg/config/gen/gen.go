package main

import (
	"github.com/conductorone/baton-sdk/pkg/config"
	cfg "github.com/conductorone/baton-beeline/pkg/config"
)

func main() {
	config.Generate("beeline", cfg.Config)
}