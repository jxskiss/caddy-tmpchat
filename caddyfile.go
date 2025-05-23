package tmpchat

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("tmpchat", parseCaddyfile)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (p *TempChat) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume directive name

	if d.Next() {
		return d.Err("expect no arguments")
	}
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new TempChat.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m = &TempChat{}
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}
