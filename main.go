package main

import (
	// Caddy
	"github.com/caddyserver/caddy/caddy/caddymain"

	// Plugins
	_ "github.com/argylelabcoat/caddy-supervisor/httpplugin"
	_ "github.com/argylelabcoat/caddy-supervisor/servertype"
	_ "github.com/hacdias/caddy-service"
)

func main() {
	caddymain.Run()
}
