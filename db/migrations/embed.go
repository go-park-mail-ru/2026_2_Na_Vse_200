// Package migrations содержит SQL-миграции, встраиваемые в команду migrate.
package migrations

import "embed"

// Files содержит исходные SQL-файлы без генерации или преобразования DDL.
//
//go:embed *.sql
var Files embed.FS
