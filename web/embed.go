package web

import "embed"

// Assets embeds the entire static web frontend into the Go binary.
//
//go:embed index.html css/* js/* js/ui/* assets/cards/*
var Assets embed.FS
