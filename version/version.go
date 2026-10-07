package version

import "runtime/debug"

const Framework = "4.0.0-dev.1"
const StableModule = "v4.0.0"
const DocsURL = "https://github.com/arfajhf/copytygo/tree/main/docs"

func Module() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return StableModule
}
