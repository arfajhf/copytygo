package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/arfajhf/copytygo/v4/branding"
)

func TestAuthUpgradeRestoresPublicFolderAndLogo(t *testing.T) {
	root := authProject(t)
	// Reproduce a project generated/cloned before PNG branding was copied to public.
	if err := os.RemoveAll(filepath.Join(root, "frontend/public")); err != nil {
		t.Fatal(err)
	}
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	logo, err := os.ReadFile(filepath.Join(root, "frontend/public/logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(logo, branding.Logo) {
		t.Fatal("default logo not restored exactly")
	}
	for _, path := range []string{"frontend/public/images", "frontend/public/images/.gitkeep", "frontend/ASSETS.md"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatal(err)
		}
	}
	customLogo := []byte("custom application logo")
	os.WriteFile(filepath.Join(root, "frontend/public/logo.png"), customLogo, 0644)
	os.WriteFile(filepath.Join(root, "frontend/public/images/banner.png"), []byte("custom image"), 0644)
	if err := InstallAuth("multi", root); err != nil {
		t.Fatal(err)
	}
	logo, _ = os.ReadFile(filepath.Join(root, "frontend/public/logo.png"))
	if !bytes.Equal(logo, customLogo) {
		t.Fatal("custom logo replaced")
	}
	image, _ := os.ReadFile(filepath.Join(root, "frontend/public/images/banner.png"))
	if string(image) != "custom image" {
		t.Fatal("custom public asset replaced")
	}
}
func TestPublicAssetRepairDoesNotNeedReinstall(t *testing.T) {
	root := t.TempDir()
	if err := ensurePublicAssets(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "frontend/public/logo.png")); err != nil {
		t.Fatal(err)
	}
	if err := ensurePublicAssets(root); err != nil {
		t.Fatal(err)
	}
	logo, _ := os.ReadFile(filepath.Join(root, "frontend/public/logo.png"))
	if !bytes.Equal(logo, branding.Logo) {
		t.Fatal("missing logo was not repaired")
	}
}
