package storage

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func uploadedFileHeader(t *testing.T, filename string, data []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("attachment", filename)
	if err != nil { t.Fatal(err) }
	if _, err := part.Write(data); err != nil { t.Fatal(err) }
	if err := writer.Close(); err != nil { t.Fatal(err) }
	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1<<20); err != nil { t.Fatal(err) }
	t.Cleanup(func() { if req.MultipartForm != nil { _ = req.MultipartForm.RemoveAll() } })
	file, header, err := req.FormFile("attachment")
	if err != nil { t.Fatal(err) }
	if err := file.Close(); err != nil { t.Fatal(err) }
	return header
}

func TestSaveUploadedFileAllowedSniffsContentInsteadOfFilename(t *testing.T) {
	disk, err := NewLocal(t.TempDir())
	if err != nil { t.Fatal(err) }
	png := append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, []byte("content")...)
	header := uploadedFileHeader(t, "untrusted.txt", png)
	if err := SaveUploadedFileAllowed(disk, "private/avatar.png", header, 1024, "image/png"); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if !disk.Exists("private/avatar.png") { t.Fatal("valid upload not stored") }

	fake := uploadedFileHeader(t, "fake.png", []byte("<html><script>alert(1)</script></html>"))
	if err := SaveUploadedFileAllowed(disk, "private/bad.png", fake, 1024, "image/png"); err == nil {
		t.Fatal("HTML with PNG extension was accepted")
	}
	if disk.Exists("private/bad.png") { t.Fatal("disallowed upload was persisted") }
}

func TestSaveUploadedFileAllowedRejectsUnboundedAndInvalidPolicies(t *testing.T) {
	disk, err := NewLocal(t.TempDir())
	if err != nil { t.Fatal(err) }
	header := uploadedFileHeader(t, "photo.png", []byte{137, 80, 78, 71, 13, 10, 26, 10})
	tests := []struct {
		name string
		limit int64
		types []string
	}{
		{"zero size limit", 0, []string{"image/png"}},
		{"negative size limit", -1, []string{"image/png"}},
		{"no MIME allowlist", 10, nil},
		{"wildcard MIME", 10, []string{"image/*"}},
		{"invalid second MIME", 10, []string{"image/png", "*/*"}},
		{"too small", 7, []string{"image/png"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := "private/" + tc.name + ".png"
			if err := SaveUploadedFileAllowed(disk, path, header, tc.limit, tc.types...); err == nil {
				t.Fatal("unsafe upload settings were accepted")
			}
			if disk.Exists(path) {
				t.Fatal("upload was written despite invalid security policy")
			}
		})
	}
}
