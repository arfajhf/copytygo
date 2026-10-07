package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GeneratedResource struct {
	Resource       string `json:"resource"`
	ControllerType string `json:"controller"`
	File           string `json:"file"`
}

func DiscoverGeneratedResources(root string) ([]GeneratedResource, error) {
	controllerDir := filepath.Join(root, "app", "controllers")
	entries, err := os.ReadDir(controllerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []GeneratedResource{}, nil
		}
		return nil, fmt.Errorf("copytygo: unable to read controllers: %w", err)
	}

	resources := make([]GeneratedResource, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}

		path := filepath.Join(controllerDir, entry.Name())
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(file)
		resource := ""
		controllerType := ""
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if matches := resourceMarkerPattern.FindStringSubmatch(line); len(matches) == 2 {
				resource = matches[1]
			}
			if strings.HasPrefix(line, "type ") && strings.Contains(line, "Controller struct") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					controllerType = fields[1]
				}
			}
		}
		_ = file.Close()

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		if resource != "" && controllerType != "" {
			resources = append(resources, GeneratedResource{
				Resource:       resource,
				ControllerType: controllerType,
				File:           entry.Name(),
			})
		}
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Resource < resources[j].Resource
	})
	return resources, nil
}
