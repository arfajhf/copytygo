# Images and public assets

The public folder for a CopyTyGo frontend is **`frontend/public/`**. New projects include the supplied `logo.png` and an `images/` directory. Keep TypeScript UI code in `frontend/src/` and put assets that visitors may access directly in `frontend/public/`.

| File you create | URL in the browser or TypeScript |
| --- | --- |
| `frontend/public/logo.png` | `/logo.png` |
| `frontend/public/images/banner.png` | `/images/banner.png` |
| `frontend/public/images/products/book.webp` | `/images/products/book.webp` |
| `frontend/public/fonts/inter.woff2` | `/fonts/inter.woff2` |
| `frontend/public/photo.jpg` | `/photo.jpg` |

Do not include `frontend/public` or `public` in the URL. Use the application URL printed by `ctg dev`, for example `http://127.0.0.1:8080/images/banner.png`.

In TypeScript:

```ts
const image = document.createElement("img");
image.src = "/images/banner.png";
image.alt = "Application banner";
document.querySelector("#app")?.append(image);
```

In CSS:

```css
.hero {
  background-image: url("/images/banner.png");
}
```

The native Go server serves public files alongside your JSON APIs. New files become available while `ctg dev` is running. Application routes take precedence, and `/api` and `/__copytygo` stay reserved for backend/Studio. Directory listings, hidden files and paths escaping the public root are unavailable. Keep private uploads outside this folder.

`ctg build` copies public assets through Vite into `frontend/dist/` and packages them in `build/frontend/dist/`. Deploy the complete `build/` folder and start the application from it, with production environment variables configured. Production serves compiled assets without Node.js.

## Repair a project upgraded from an older starter

```powershell
go install github.com/arfajhf/copytygo/v4/cmd/ctg@v4.0.6
go get github.com/arfajhf/copytygo/v4@v4.0.6
go mod tidy
ctg install:auth multi
ctg dev
```

Use `single` if that was the original auth mode. Reinstallation adds missing public folders and restores the supplied logo if `frontend/public/logo.png` is missing. Existing logos, images, customized TypeScript and database migrations remain intact. For an already installed TypeScript auth project, `ctg dev` and `ctg build` also repair missing default assets automatically.

For a custom Go application, register static serving explicitly when needed:

```go
app.Static("/images", "path/to/public/images")
```

Static serving supports GET/HEAD, standard content types and range requests, runs global middleware, and checks application routes first.
