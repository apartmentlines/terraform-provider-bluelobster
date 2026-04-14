package main

import (
	"context"
	_ "embed"
	"flag"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/apartmentlines/terraform-provider-bluelobster/internal/provider"
)

//go:embed VERSION
var version string

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
	flag.Parse()

	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/apartmentlines/bluelobster",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
