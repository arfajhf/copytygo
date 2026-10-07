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

	_, header, err := ctx.Request.FormFile(name)
	if err != nil {
		if err == http.ErrMissingFile {
			return nil, NewHTTPError(http.StatusBadRequest, fmt.Sprintf("File %q is required", name))
		}
		return nil, NewHTTPError(http.StatusBadRequest, "Unable to read uploaded file")
	}

	return header, nil
}
