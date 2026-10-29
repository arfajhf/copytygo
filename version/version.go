package version

import "runtime/debug"

const Framework = "5.0.0"
const StableModule = "v5.0.0"
const DocsURL = "https://github.com/arfajhf/copytygo/tree/main/docs"

func Module() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return StableModule
}
