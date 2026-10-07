package cli

import (
	"fmt"
	"io"
)

func devWithRuntimes(args []string, native, lite func() error, output io.Writer) error {
	forceLite, forceNative := false, false
	for _, arg := range args {
		switch arg {
		case "--lite":
			forceLite = true
		case "--native":
			forceNative = true
		default:
			return fmt.Errorf("usage: ctg dev [--lite|--native]")
		}
	}
	if forceLite && forceNative {
		return fmt.Errorf("copytygo: --lite and --native cannot be combined")
	}
	if forceLite {
		return lite()
	}
	// Always run the full application. Decisions cached by older CLI versions
	// must not keep a project in Lite after a transient execution-policy block.
	fmt.Fprintln(output, "CopyTyGo Dev\n-------------\nRuntime : native")
	fmt.Fprintln(output)
	return native()
}
