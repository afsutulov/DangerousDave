// Package assets embeds the artwork and level data recovered from the supplied DOS copy.
package assets

import "embed"

// FS contains data only; no DOS executable is embedded.
//
//go:embed *.png *.json *.bin
var FS embed.FS
