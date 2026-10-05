package core

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

func (ctx *Context) Bind(target any) error {
	contentType := ctx.Request.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil && contentType != "" {
		return NewHTTPError(
			http.StatusBadRequest,
			"Invalid Content-Type",
		)
	}

	switch mediaType {

	case "application/json":
		return ctx.bindJSON(target)

	case "application/x-www-form-urlencoded":
		return ctx.bindForm(target)

	case "multipart/form-data":
		return ctx.bindForm(target)

	default:
		if contentType == "" {
			return NewHTTPError(
				http.StatusUnsupportedMediaType,
				"Content-Type is required",
			)
		}

		return NewHTTPError(
			http.StatusUnsupportedMediaType,
			"Unsupported Content-Type",
		)
	}
}

func (ctx *Context) bindJSON(target any) error {
	decoder := json.NewDecoder(ctx.Request.Body)

	err := decoder.Decode(target)

	if errors.Is(err, io.EOF) {
		return NewHTTPError(
			http.StatusBadRequest,
			"Request body cannot be empty",
		)
	}

	if err != nil {
		return NewHTTPError(
			http.StatusBadRequest,
			"Invalid JSON body",
		)
	}

	var extra any

	if err := decoder.Decode(&extra); err != io.EOF {
		return NewHTTPError(
			http.StatusBadRequest,
			"Request body must contain only one JSON value",
		)
	}

	return nil
}

func (ctx *Context) bindForm(target any) error {
	if err := ctx.Request.ParseForm(); err != nil {
		return NewHTTPError(
			http.StatusBadRequest,
			"Invalid form body",
		)
	}

	values := make(map[string]string)

	for key, value := range ctx.Request.Form {
		if len(value) > 0 {
			values[key] = value[0]
		}
	}

	raw, err := json.Marshal(values)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(strings.NewReader(string(raw)))

	if err := decoder.Decode(target); err != nil {
		return NewHTTPError(
			http.StatusBadRequest,
			"Unable to bind form data",
		)
	}

	return nil
}
