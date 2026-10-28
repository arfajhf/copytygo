package cli

import (
	"reflect"
	"testing"
)

func TestParseResourceArguments(t *testing.T) {
	tests := []struct {
		args []string
		fields []string
		access string
		ok bool
	}{
		{[]string{"name:string"}, []string{"name:string"}, "default", true},
		{[]string{"name:string", "--public"}, []string{"name:string"}, "public", true},
		{[]string{"--access=admin", "name:string"}, []string{"name:string"}, "admin", true},
		{[]string{"--access=authenticated"}, nil, "authenticated", true},
		{[]string{"--access=everyone"}, nil, "", false},
		{[]string{"--public", "--access=admin"}, nil, "", false},
		{[]string{"--no-auth"}, nil, "", false},
	}
	for _, tc := range tests {
		fields, access, err := parseResourceArguments(tc.args)
		if (err == nil) != tc.ok {
			t.Fatalf("args %q: err=%v", tc.args, err)
		}
		if tc.ok && (!reflect.DeepEqual(fields, tc.fields) || access != tc.access) {
			t.Fatalf("args %q: got %q/%q", tc.args, fields, access)
		}
	}
}
