package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
				fmt.Printf("%-8s %s\n", m[1], m[2])
			}
		}
		return scanner.Err()
	})
}
