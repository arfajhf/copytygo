package cli

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestEcosystemGeneratorsProduceValidGo(t *testing.T) {
	root := t.TempDir()

	cases := []struct {
		name string
		path string
		make func() error
	}{
		{
			name: "job",
			path: filepath.Join(root, "jobs", "sendemail.go"),
			make: func() error { return MakeJob("SendEmail", filepath.Join(root, "jobs")) },
		},
		{
			name: "listener",
			path: filepath.Join(root, "listeners", "notifyadmin.go"),
			make: func() error { return MakeListener("NotifyAdmin", filepath.Join(root, "listeners")) },
		},
		{
			name: "seeder",
			path: filepath.Join(root, "seeders", "userseeder.go"),
			make: func() error { return MakeSeeder("UserSeeder", filepath.Join(root, "seeders")) },
		},
		{
			name: "factory",
			path: filepath.Join(root, "factories", "user.go"),
			make: func() error { return MakeFactory("User", filepath.Join(root, "factories")) },
		},
		{
			name: "mail",
			path: filepath.Join(root, "mails", "welcome.go"),
			make: func() error { return MakeMail("Welcome", filepath.Join(root, "mails")) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.make(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(tc.path); err != nil {
				t.Fatalf("expected generated file: %v", err)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), tc.path, nil, parser.AllErrors); err != nil {
				t.Fatalf("generated Go is invalid: %v", err)
			}
		})
	}
}
