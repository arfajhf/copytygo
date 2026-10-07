package version

import "runtime/debug"

const Framework = "4.0.3"
const StableModule = "v4.0.3"
const DocsURL = "https://github.com/arfajhf/copytygo/tree/v4.0.3/docs"

func Module() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return StableModule
}
