package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func InteractiveNewProject(initialName string) error {
	return interactiveNewProject(os.Stdin, os.Stdout, initialName)
}

func interactiveNewProject(input io.Reader, output io.Writer, initialName string) error {
	reader := bufio.NewReader(input)

	name := strings.TrimSpace(initialName)
	if name == "" {
		value, err := prompt(reader, output, "Project name", "my-app")
		if err != nil {
			return err
		}
		name = value
	}

	database, err := promptChoice(reader, output, "Database", "mysql", []string{"mysql", "postgres"})
	if err != nil {
		return err
	}

	auth, err := promptChoice(reader, output, "Authentication", "none", []string{"none", "single", "multi"})
	if err != nil {
		return err
	}

	frontend, err := promptChoice(reader, output, "Frontend", "typescript", []string{"typescript", "react", "vue", "api"})
	if err != nil {
		return err
	}

	studioValue, err := promptChoice(reader, output, "Install CopyTyGo Studio", "yes", []string{"yes", "no"})
	if err != nil {
		return err
	}

	fmt.Fprintln(output)
	fmt.Fprintln(output, "Creating application...")
	fmt.Fprintln(output)

	return NewProjectWithOptions(name, ProjectOptions{
		Database: database,
		Auth:     auth,
		Frontend: frontend,
		Studio:   studioValue == "yes",
	})
}

func prompt(reader *bufio.Reader, output io.Writer, label, fallback string) (string, error) {
	fmt.Fprintf(output, "? %s [%s]: ", label, fallback)
	raw, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		value = fallback
	}
	return value, nil
}

func promptChoice(
	reader *bufio.Reader,
	output io.Writer,
	label string,
	fallback string,
	choices []string,
) (string, error) {
	for {
		fmt.Fprintf(output, "? %s (%s) [%s]: ", label, strings.Join(choices, "/"), fallback)
		raw, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			return fallback, nil
		}
		for _, choice := range choices {
			if value == choice {
				return value, nil
			}
		}
		fmt.Fprintf(output, "  Choose one of: %s\n", strings.Join(choices, ", "))
		if err == io.EOF {
			return fallback, nil
		}
	}
}
