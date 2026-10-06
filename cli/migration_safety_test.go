package cli

import "testing"

func TestProductionDestructiveMigrationRequiresForce(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	if err := requireProductionForce(nil, "migrate:fresh"); err == nil {
		t.Fatal("expected production force requirement")
	}

	if err := requireProductionForce([]string{"--force"}, "migrate:fresh"); err != nil {
		t.Fatalf("expected --force to allow destructive migration: %v", err)
	}
}

func TestLocalDestructiveMigrationDoesNotRequireForce(t *testing.T) {
	t.Setenv("APP_ENV", "local")

	if err := requireProductionForce(nil, "migrate:fresh"); err != nil {
		t.Fatalf("did not expect force requirement in local env: %v", err)
	}
}
