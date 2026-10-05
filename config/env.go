package config

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

func LoadEnv(path string) error {
	file, err := os.Open(path)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")

		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "" {
			continue
		}

		value = cleanEnvValue(value)

		// Environment variable asli memiliki prioritas.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func cleanEnvValue(value string) string {
	if len(value) < 2 {
		return value
	}

	doubleQuoted :=
		strings.HasPrefix(value, `"`) &&
			strings.HasSuffix(value, `"`)

	singleQuoted :=
		strings.HasPrefix(value, `'`) &&
			strings.HasSuffix(value, `'`)

	if doubleQuoted || singleQuoted {
		return value[1 : len(value)-1]
	}

	return value
}
