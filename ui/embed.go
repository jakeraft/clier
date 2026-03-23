package ui

import "embed"

//go:embed all:dist
var DistFS embed.FS

const DistRoot = "dist"
