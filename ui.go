// Package konsul main package
package konsul

import "embed"

// AdminUI is the embedded admin UI
//
//go:embed web/admin/dist
var AdminUI embed.FS
