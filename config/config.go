package config

import (
	"os"
	"strconv"
)

func Get(key string, defaults ...string) string {
	value, exists := os.LookupEnv(key)

	if exists {
		return value
	}

	if len(defaults) > 0 {
		return defaults[0]
	}

	return ""
}

func GetInt(key string, defaults ...int) int {
	value := Get(key)

	if value == "" {
		if len(defaults) > 0 {
			return defaults[0]
		}

		return 0
	}

	number, err := strconv.Atoi(value)

	if err != nil {
		if len(defaults) > 0 {
			return defaults[0]
		}

		return 0
	}

	return number
}

func GetBool(key string, defaults ...bool) bool {
	value := Get(key)

	if value == "" {
		if len(defaults) > 0 {
			return defaults[0]
		}

		return false
	}

	boolean, err := strconv.ParseBool(value)

	if err != nil {
		if len(defaults) > 0 {
			return defaults[0]
		}

		return false
	}

	return boolean
}
