package web

import "embed"

//go:embed all:explorer/out
var ExplorerFS embed.FS

//go:embed all:wallet/out
var WalletFS embed.FS
