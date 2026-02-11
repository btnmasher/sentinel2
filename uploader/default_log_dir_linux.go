//go:build linux

package main

func defaultLogDir() string {
	// Linux installs vary (Wine/Proton). Require explicit log directory.
	return ""
}
