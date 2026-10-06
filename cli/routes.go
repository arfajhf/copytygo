package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var routePattern = regexp.MustCompile(`\.(Get|Post|Put|Patch|Delete)\("([^"]+)"`)

func RouteList(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			m := routePattern.FindStringSubmatch(scanner.Text())
			if len(m) == 3 {
				fmt.Printf("%-8s %s\n", strings.ToUpper(m[1]), m[2])
			}

			resource := resourceRoutePattern.FindStringSubmatch(scanner.Text())
			if len(resource) == 2 {
				base := strings.TrimRight(resource[1], "/")
				if base == "" {
					base = "/"
				}
				member := base
				if member == "/" {
					member = ""
				}
				member += "/:id"

				fmt.Printf("%-8s %s\n", "GET", base)
				fmt.Printf("%-8s %s\n", "GET", member)
				fmt.Printf("%-8s %s\n", "POST", base)
				fmt.Printf("%-8s %s\n", "PUT", member)
				fmt.Printf("%-8s %s\n", "DELETE", member)
			}
		}
		return scanner.Err()
	})
}
