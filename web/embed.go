// Package web embute o frontend compilado (web/dist).
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
