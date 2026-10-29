package core

import (
	"fmt"
	"mime/multipart"
	"net/http"
)

const defaultMultipartMemory = int64(32 << 20)

func (ctx *Context) File(name string, maxMemory ...int64) (*multipart.FileHeader, error) {
	if ctx == nil || ctx.Request == nil {
		return nil, NewHTTPError(http.StatusBadRequest, "Request is unavailable")
	}

	limit := defaultMultipartMemory
	if len(maxMemory) > 0 && maxMemory[0] > 0 {
		limit = maxMemory[0]
	}

	if err := ctx.Request.ParseMultipartForm(limit); err != nil {
		return nil, NewHTTPError(http.StatusBadRequest, "Invalid multipart form")
	}

	file, header, err := ctx.Request.FormFile(name)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, NewHTTPError(http.StatusBadRequest, fmt.Sprintf("File %q is required", name))
		}
		return nil, NewHTTPError(http.StatusBadRequest, "Unable to read uploaded file")
	}

	// FormFile opens a reader even though Context.File only returns the header.
	// Close that descriptor; callers can open the header again when saving.
	if err := file.Close(); err != nil {
		return nil, NewHTTPError(http.StatusBadRequest, "Unable to close uploaded file")
	}
	return header, nil
}
