package assets

import "embed"

//go:embed "emails"
var EmailFS embed.FS

//go:embed "migrations"
var MigrationFS embed.FS
