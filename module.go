package tmpchat

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(TempChat{})
}

type TempChat struct {
	server *ChatServer
}

// CaddyModule returns the Caddy module information.
func (TempChat) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.tmpchat",
		New: func() caddy.Module { return new(TempChat) },
	}
}

// Provision implements caddy.Provisioner.
func (p *TempChat) Provision(ctx caddy.Context) error {
	p.server = NewChatServer()
	p.server.logger = ctx.Logger()
	go p.server.Run()
	return nil
}

// Validate implements caddy.Validator
func (p *TempChat) Validate() error { return nil }

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (p *TempChat) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {
	// Serve websocket request
	if r.Header.Get("Upgrade") != "" {
		p.server.serveWebsocket(w, r)
		return nil
	}

	// Serve the page request
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(chatPage)
	return nil
}

// Cleanup implements caddy.Cleanup
func (p *TempChat) Cleanup() error {
	p.server.Close()
	return nil
}
