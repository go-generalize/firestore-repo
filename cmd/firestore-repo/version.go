package main

import "runtime/debug"

func getAppVersion() string {
	bi, ok := debug.ReadBuildInfo()

	if !ok {
		return "devel"
	}

	return bi.Main.Version
}
