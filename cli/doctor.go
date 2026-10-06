package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v3/security"
	"github.com/arfajhf/copytygo/v3/version"
)

type doctorCheck struct {
	Name string
	Err  error
}

func Doctor(args []string) error {
	fmt.Printf("CopyTyGo Doctor v%s\n", version.Framework)
	fmt.Println("------------------------------")

	checks := []doctorCheck{
		{Name: "go.mod", Err: doctorFileContains("go.mod", "github.com/arfajhf/copytygo/v3")},
		{Name: ".env", Err: doctorFileExists(".env")},
		{Name: "routes", Err: doctorRoutes()},
		{Name: "Lite params/query", Err: doctorLiteParams()},
		{Name: "Lite validation", Err: doctorLiteValidation()},
		{Name: "Lite CRUD memory", Err: doctorLiteCRUD()},
		{Name: "resource routes", Err: doctorResourceRoutes()},
		{Name: "migration registry", Err: doctorMigrationRegistry()},
		{Name: "security", Err: doctorSecurity()},
	}

	for _, arg := range args {
		if arg == "--db" {
			checks = append(checks, doctorCheck{Name: "database", Err: databaseHealthCheck()})
			break
		}
	}

	failed := 0
	for _, check := range checks {
		if check.Err != nil {
			failed++
			fmt.Printf("x %-20s %v\n", check.Name, check.Err)
			continue
		}
		fmt.Printf("✓ %s\n", check.Name)
	}

	fmt.Println()
	if failed > 0 {
		return fmt.Errorf("copytygo doctor: %d check(s) failed", failed)
	}

	fmt.Println("All CopyTyGo checks passed.")
	return nil
}

func doctorFileExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return nil
}

func doctorFileContains(path, needle string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(raw), needle) {
		return fmt.Errorf("%s does not reference CopyTyGo", path)
	}
	return nil
}

func doctorRoutes() error {
	routes, err := discoverLiteRoutes("routes")
	if err != nil {
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("no Lite-compatible routes found")
	}
	return nil
}

func doctorLiteParams() error {
	params, ok := matchLitePath("/doctor/:id", "/doctor/42")
	if !ok || params["id"] != "42" {
		return fmt.Errorf("route parameter matching failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/doctor/42?q=ok", nil)
	body := renderLiteBody("{{param:id}}-{{query:q}}", req, params)
	if body != "42-ok" {
		return fmt.Errorf("request rendering failed")
	}
	return nil
}

func doctorLiteValidation() error {
	req := httptest.NewRequest(
		http.MethodPost,
		"/doctor",
		strings.NewReader(`{"name":"A","email":"wrong"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	fields := liteValidateRequest(req, map[string]string{
		"name":  "required|min:3",
		"email": "required|email",
	})
	if len(fields["name"]) == 0 || len(fields["email"]) == 0 {
		return fmt.Errorf("validation rules did not execute")
	}
	return nil
}

func doctorLiteCRUD() error {
	resource := "__copytygo_doctor"
	liteMemoryStore.Lock()
	delete(liteMemoryStore.Buckets, resource)
	liteMemoryStore.Unlock()

	store := liteRoute{
		ResourceAction: "store",
		Resource:       resource,
		ResourceData:   `{"name":"{{input:name}}"}`,
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/doctor/resources",
		strings.NewReader(`{"name":"CopyTyGo"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleLiteMemoryRoute(rec, req, store, nil)
	if rec.Code != http.StatusCreated {
		return fmt.Errorf("create returned %d", rec.Code)
	}

	show := httptest.NewRecorder()
	handleLiteMemoryRoute(
		show,
		httptest.NewRequest(http.MethodGet, "/doctor/resources/1", nil),
		liteRoute{
			ResourceAction: "show",
			Resource:       resource,
			ResourceID:     "{{param:id}}",
		},
		map[string]string{"id": "1"},
	)
	if show.Code != http.StatusOK || !strings.Contains(show.Body.String(), "CopyTyGo") {
		return fmt.Errorf("show failed")
	}

	remove := httptest.NewRecorder()
	handleLiteMemoryRoute(
		remove,
		httptest.NewRequest(http.MethodDelete, "/doctor/resources/1", nil),
		liteRoute{
			ResourceAction: "destroy",
			Resource:       resource,
			ResourceID:     "{{param:id}}",
		},
		map[string]string{"id": "1"},
	)
	if remove.Code != http.StatusNoContent {
		return fmt.Errorf("delete returned %d", remove.Code)
	}

	return nil
}


func doctorResourceRoutes() error {
	if _, err := os.Stat(filepath.Join("routes", "resources.go")); err != nil {
		// Projects without generated resources are still healthy.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	raw, err := os.ReadFile(filepath.Join("routes", "resources.go"))
	if err != nil {
		return err
	}
	if !strings.Contains(string(raw), "RegisterGeneratedResources") {
		return fmt.Errorf("generated resource route registry is invalid")
	}
	return nil
}

func doctorMigrationRegistry() error {
	path := filepath.Join("database", "migrations")
	if _, err := os.Stat(path); err != nil {
		return err
	}

	registry := filepath.Join(path, "registry.go")
	if _, err := os.Stat(registry); err != nil {
		// Empty projects may not have migrations yet.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	raw, err := os.ReadFile(registry)
	if err != nil {
		return err
	}
	if !strings.Contains(string(raw), "RegisterGenerated") {
		return fmt.Errorf("migration registry is invalid")
	}
	return nil
}

func doctorSecurity() error {
	hash, err := security.HashPassword("copytygo-doctor-password")
	if err != nil {
		return err
	}
	if !security.VerifyPassword("copytygo-doctor-password", hash) {
		return fmt.Errorf("password hashing verification failed")
	}

	token, err := security.SignToken("doctor-secret", security.Claims{
		Subject:   "doctor",
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
	})
	if err != nil {
		return err
	}
	claims, err := security.VerifyToken("doctor-secret", token)
	if err != nil || claims.Subject != "doctor" {
		return fmt.Errorf("token verification failed")
	}
	return nil
}
