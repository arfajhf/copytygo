package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arfajhf/copytygo/v4/branding"
)

// ensurePublicAssets repairs older projects as well as scaffolding new ones.
// Existing assets, including a customized logo, always belong to the application.
func ensurePublicAssets(root string) error {
	directory := filepath.Join(root, "frontend", "public")
	if err := os.MkdirAll(filepath.Join(directory, "images"), 0755); err != nil {
		return err
	}
	for path, data := range map[string][]byte{
		filepath.Join(directory, "logo.png"):           branding.Logo,
		filepath.Join(directory, "images", ".gitkeep"): nil,
		filepath.Join(root, "frontend", "ASSETS.md"):   []byte(publicAssetsGuide),
	} {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("copytygo public asset %s: %w", path, err)
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

const publicAssetsGuide = `# Public assets

Put images and other static assets in frontend/public/.
The starter includes frontend/public/logo.png and frontend/public/images/.

Examples:

| File | URL / TypeScript reference |
| --- | --- |
| frontend/public/logo.png | /logo.png |
| frontend/public/images/banner.png | /images/banner.png |
| frontend/public/images/products/book.webp | /images/products/book.webp |
| frontend/public/fonts/inter.woff2 | /fonts/inter.woff2 |

Use the URL in your TypeScript UI, for example:

const image = document.createElement("img");
image.src = "/images/banner.png";
image.alt = "Banner";

Do not include /frontend/public/ or /public/ in browser URLs.
Use the Go application URL printed by ctg dev. New public files work immediately.
Public files are accessible to visitors; store private uploads elsewhere.
API and Studio URLs stay reserved for backend routes. Directory listings,
hidden files and paths escaping the public directory are unavailable.

ctg build copies public assets into frontend/dist/ and packages the output in
build/frontend/dist/. Deploy the complete build folder and start the application
from that folder. Node.js is not needed to serve compiled production assets.

ctg install:auth, ctg dev and ctg build restore the default logo if it is missing.
A logo you have already supplied is preserved. Replace frontend/public/logo.png
to use your own branding, then rebuild for production.
`
