// Package lobstermigrations embeds the SQL migration files into the binary.
// This allows lobster to run migrations regardless of the process working directory.
package lobstermigrations

import "embed"

// FS contains all migration SQL files, embedded at compile time.
//
//go:embed *.sql
var FS embed.FS
