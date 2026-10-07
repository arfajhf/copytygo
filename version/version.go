package version

import "runtime/debug"

const Framework = "4.0.1"
const StableModule = "v4.0.1"
const DocsURL = "https://github.com/arfajhf/copytygo/tree/v4.0.1/docs"

func Module() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return StableModule
}
