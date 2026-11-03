package migration

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"2026_2_Na_Vse_200/db/migrations"
)

func TestParseCommand(t *testing.T) {
	for _, tc := range []struct {
		args   []string
		action string
		steps  int
	}{
		{[]string{"up"}, "up", 0},
		{[]string{"version"}, "version", 0},
		{[]string{"down", "2"}, "down", 2},
		{[]string{"-h"}, "-h", 0},
		{[]string{"--help"}, "--help", 0},
	} {
		cmd, err := parseCommand(tc.args)
		if err != nil || cmd.action != tc.action || cmd.steps != tc.steps {
			t.Fatalf("%v: %+v, %v", tc.args, cmd, err)
		}
	}
	for _, args := range [][]string{nil, {"bad"}, {"up", "1"}, {"version", "1"}, {"--help", "extra"}, {"down"}, {"down", "0"}, {"down", "-1"}, {"down", "text"}, {"down", "1", "2"}} {
		if _, err := parseCommand(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestHelpWithoutDatabase(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "")
	if err := Run(context.Background(), []string{"--help"}); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidConnectionSettings(t *testing.T) {
	for _, dsn := range []string{"", " ", "postgres://user:secret@host:wrong-port/db"} {
		if _, err := connectDB(context.Background(), dsn); err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestMigrationBody(t *testing.T) {
	for _, query := range []string{"BEGIN; CREATE TABLE example (id int); COMMIT;", "CREATE TABLE example (id int);"} {
		got, err := migrationBody(query)
		if err != nil || got != "CREATE TABLE example (id int);" {
			t.Fatalf("body %q: %q, %v", query, got, err)
		}
	}
	for _, query := range []string{"BEGIN; SELECT 1;", "SELECT 1; COMMIT;"} {
		if _, err := migrationBody(query); err == nil {
			t.Fatalf("accepted incomplete transaction: %s", query)
		}
	}
	files, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		data, err := migrations.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		body, err := migrationBody(string(data))
		if err != nil || body == "" {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
