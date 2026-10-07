package storage

import (
	"fmt"
	"io"
	"mime/multipart"
)

func SaveUploadedFile(disk Disk, path string, header *multipart.FileHeader, maxBytes int64) error {
	if disk == nil {
		return fmt.Errorf("copytygo storage: disk is required")
	}
	if header == nil {
		return fmt.Errorf("copytygo storage: uploaded file is required")
	}
	if maxBytes > 0 && header.Size > maxBytes {
		return fmt.Errorf("copytygo storage: uploaded file exceeds %d bytes", maxBytes)
	}

	file, err := header.Open()
	if err != nil {
		return fmt.Errorf("copytygo storage: open uploaded file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file
	if maxBytes > 0 {
		reader = io.LimitReader(file, maxBytes+1)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("copytygo storage: read uploaded file: %w", err)
	}
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return fmt.Errorf("copytygo storage: uploaded file exceeds %d bytes", maxBytes)
	}

	return disk.Put(path, data)
}
