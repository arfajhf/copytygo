package version

import "runtime/debug"

const Framework = "2.0.0-rc.1"
const StableModule = "v2.0.0"

func Module() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return StableModule
}
