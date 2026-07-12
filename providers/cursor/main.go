package main

import (
	"context"
	"flag"
	"log"

	"github.com/benfdking/sdsf/providers/cursor/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "enable debugger support")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/benfdking/cursor",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
