package core

import "testing"

type resourceTestController struct{}

func (resourceTestController) Index(*Context) error   { return nil }
func (resourceTestController) Show(*Context) error    { return nil }
func (resourceTestController) Store(*Context) error   { return nil }
func (resourceTestController) Update(*Context) error  { return nil }
func (resourceTestController) Destroy(*Context) error { return nil }

func TestResourceRegistersFiveRoutes(t *testing.T) {
	app := New()
	app.Resource("/products", resourceTestController{})

	routes := app.Routes()
	if len(routes) != 5 {
		t.Fatalf("expected 5 resource routes, got %d", len(routes))
	}

	expected := []struct {
		method string
		path   string
	}{
		{"GET", "/products"},
		{"GET", "/products/:id"},
		{"POST", "/products"},
		{"PUT", "/products/:id"},
		{"DELETE", "/products/:id"},
	}

	for i, want := range expected {
		if routes[i].Method != want.method || routes[i].Path != want.path {
			t.Fatalf("route %d expected %s %s, got %s %s", i, want.method, want.path, routes[i].Method, routes[i].Path)
		}
	}
}
