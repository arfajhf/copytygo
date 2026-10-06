package core

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/arfajhf/copytygo/validation"
)

func (ctx *Context) Input(name string) string {
	if ctx.Request == nil {
		return ""
	}

	contentType := ctx.Request.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(contentType)

	switch mediaType {
	case "application/json":
		if ctx.Request.Body == nil {
			return ""
		}
		raw, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			return ""
		}
		ctx.Request.Body = io.NopCloser(strings.NewReader(string(raw)))

		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return ""
		}
		value, ok := data[name]
		if !ok || value == nil {
			return ""
		}
		switch v := value.(type) {
		case string:
			return v
		default:
			encoded, err := json.Marshal(v)
			if err != nil {
				return ""
			}
			return string(encoded)
		}
	default:
		if err := ctx.Request.ParseForm(); err != nil {
			return ""
		}
		return ctx.Request.FormValue(name)
	}
}

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


func (ctx *Context) Validate(rules map[string]string) error {
	data := make(map[string]string, len(rules))
	for field := range rules {
		data[field] = ctx.Input(field)
	}

	v := validation.New(data)

	for field, ruleList := range rules {
		for _, rule := range strings.Split(ruleList, "|") {
			rule = strings.TrimSpace(rule)
			switch {
			case rule == "required":
				v.Required(field)
			case rule == "email":
				v.Email(field)
			case rule == "integer":
				v.Integer(field)
			case strings.HasPrefix(rule, "min:"):
				if n, err := strconv.Atoi(strings.TrimPrefix(rule, "min:")); err == nil {
					v.Min(field, n)
				}
			case strings.HasPrefix(rule, "max:"):
				if n, err := strconv.Atoi(strings.TrimPrefix(rule, "max:")); err == nil {
					v.Max(field, n)
				}
			case strings.HasPrefix(rule, "oneof:"):
				values := strings.Split(strings.TrimPrefix(rule, "oneof:"), ",")
				v.OneOf(field, values...)
			}
		}
	}

	if !v.Valid() {
		return NewValidationError(map[string][]string(v.Errors()))
	}
	return nil
}
