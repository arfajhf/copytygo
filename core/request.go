package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/arfajhf/copytygo/v5/validation"
)

// Input returns a scalar form/JSON value for backward compatibility.
// Use InputWithError or Validate if malformed input must be reported as HTTP 400.
func (ctx *Context) Input(name string) string {
	value, _ := ctx.InputWithError(name)
	return value
}

// InputWithError reports malformed input instead of treating it as missing.
// The JSON decoder runs once per request and preserves JSON numeric precision.
func (ctx *Context) InputWithError(name string) (string, error) {
	values, err := ctx.readInputs()
	if err != nil {
		return "", err
	}
	return values[name], nil
}

const maxJSONInputBytes int64 = 16 << 20

func (ctx *Context) readInputs() (map[string]string, error) {
	if ctx == nil || ctx.Request == nil {
		return nil, NewHTTPError(http.StatusBadRequest, "Request is unavailable")
	}
	if ctx.inputParsed {
		return ctx.inputValues, ctx.inputError
	}
	ctx.inputParsed = true
	ctx.inputValues = make(map[string]string)

	contentType := ctx.Request.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil && contentType != "" {
		ctx.inputError = NewHTTPError(http.StatusBadRequest, "Invalid Content-Type")
		return nil, ctx.inputError
	}
	if mediaType == "application/json" {
		if ctx.Request.Body == nil {
			ctx.inputError = NewHTTPError(http.StatusBadRequest, "Invalid JSON body")
			return nil, ctx.inputError
		}
		// The generated application also installs BodyLimit(2<<20).
		// This ceiling protects manual handlers that omit that middleware.
		raw, err := io.ReadAll(io.LimitReader(ctx.Request.Body, maxJSONInputBytes+1))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				ctx.inputError = NewHTTPError(http.StatusRequestEntityTooLarge, "Request body is too large")
			} else {
				ctx.inputError = NewHTTPError(http.StatusBadRequest, "Unable to read JSON body")
			}
			return nil, ctx.inputError
		}
		if int64(len(raw)) > maxJSONInputBytes {
			ctx.inputError = NewHTTPError(http.StatusRequestEntityTooLarge, "Request body is too large")
			return nil, ctx.inputError
		}
		ctx.Request.Body = io.NopCloser(bytes.NewReader(raw))

		decoder := json.NewDecoder(bytes.NewReader(raw))
		var fields map[string]json.RawMessage
		if err := decoder.Decode(&fields); err != nil || fields == nil {
			ctx.inputError = NewHTTPError(http.StatusBadRequest, "JSON body must be an object")
			return nil, ctx.inputError
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			ctx.inputError = NewHTTPError(http.StatusBadRequest, "Request body must contain only one JSON value")
			return nil, ctx.inputError
		}
		for key, rawValue := range fields {
			value := bytes.TrimSpace(rawValue)
			if len(value) == 0 || bytes.Equal(value, []byte("null")) {
				continue
			}
			if value[0] == '"' {
				var decoded string
				if err := json.Unmarshal(value, &decoded); err != nil {
					ctx.inputError = NewHTTPError(http.StatusBadRequest, "Invalid JSON string")
					return nil, ctx.inputError
				}
				ctx.inputValues[key] = decoded
			} else {
				ctx.inputValues[key] = string(value)
			}
		}
		return ctx.inputValues, nil
	}
	if err := ctx.Request.ParseForm(); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			ctx.inputError = NewHTTPError(http.StatusRequestEntityTooLarge, "Request body is too large")
		} else {
			ctx.inputError = NewHTTPError(http.StatusBadRequest, "Invalid form body")
		}
		return nil, ctx.inputError
	}
	for name, values := range ctx.Request.Form {
		if len(values) > 0 {
			ctx.inputValues[name] = values[0]
		}
	}
	return ctx.inputValues, nil
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
	data, err := ctx.readInputs()
	if err != nil {
		return err
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
			case rule == "numeric":
				v.Numeric(field)
			case rule == "boolean":
				v.Boolean(field)
			case rule == "uuid":
				v.UUID(field)
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
