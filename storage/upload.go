package storage

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
)

// SaveUploadedFile keeps existing behavior for backwards compatibility.
// Callers must set maxBytes > 0 to enforce a file-size limit and validate
// content types separately when accepting untrusted uploads.
func SaveUploadedFile(disk Disk, path string, header *multipart.FileHeader, maxBytes int64) error {
	if disk == nil {
		return fmt.Errorf("copytygo storage: disk is required")
	}
	data, err := readUploadedFile(header, maxBytes)
	if err != nil {
		return err
	}
	return disk.Put(path, data)
}

// SaveUploadedFileAllowed accepts only explicitly whitelisted, content-sniffed
// MIME types and requires a positive size limit. Filename extensions and client
// Content-Type headers are not trusted. Application code still decides which
// users may upload, where private files are kept, and how files are served.
func SaveUploadedFileAllowed(disk Disk, path string, header *multipart.FileHeader, maxBytes int64, allowedTypes ...string) error {
	if disk == nil {
		return fmt.Errorf("copytygo storage: disk is required")
	}
	if maxBytes <= 0 {
		return fmt.Errorf("copytygo storage: positive upload size limit is required")
	}
	if len(allowedTypes) == 0 {
		return fmt.Errorf("copytygo storage: at least one allowed MIME type is required")
	}
	data, err := readUploadedFile(header, maxBytes)
	if err != nil {
		return err
	}
	detected, _, err := mime.ParseMediaType(http.DetectContentType(data))
	if err != nil {
		return fmt.Errorf("copytygo storage: unable to detect uploaded content type: %w", err)
	}
	matched := false
	for _, allowed := range allowedTypes {
		expected, _, err := mime.ParseMediaType(strings.TrimSpace(allowed))
		if err != nil || expected == "" || strings.Contains(expected, "*") {
			return fmt.Errorf("copytygo storage: invalid allowed MIME type %q", allowed)
		}
		if strings.EqualFold(detected, expected) {
			matched = true
		}
	}
	if !matched {
		return fmt.Errorf("copytygo storage: uploaded content type %q is not allowed", detected)
	}
	return disk.Put(path, data)
}

func readUploadedFile(header *multipart.FileHeader, maxBytes int64) ([]byte, error) {
	if header == nil {
		return nil, fmt.Errorf("copytygo storage: uploaded file is required")
	}
	if maxBytes > 0 && header.Size > maxBytes {
		return nil, fmt.Errorf("copytygo storage: uploaded file exceeds %d bytes", maxBytes)
	}
	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("copytygo storage: open uploaded file: %w", err)
	}
	defer file.Close()
	var reader io.Reader = file
	if maxBytes > 0 {
		reader = io.LimitReader(file, maxBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("copytygo storage: read uploaded file: %w", err)
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("copytygo storage: uploaded file exceeds %d bytes", maxBytes)
	}
	return data, nil
}
