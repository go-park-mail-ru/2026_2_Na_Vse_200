package migrations

import "embed"

// Files содержит SQL-файлы миграций.
//
//go:embed *.sql
var Files embed.FS
