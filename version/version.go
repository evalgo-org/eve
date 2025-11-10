// Package version provides utilities for extracting build and dependency information
package version

import (
	"runtime/debug"
	"sort"
)

// DependencyInfo represents a module dependency and its version
type DependencyInfo struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Replace string `json:"replace,omitempty"` // If module is replaced
}

// BuildInfo contains build-time information
type BuildInfo struct {
	GoVersion    string           `json:"goVersion"`
	MainModule   string           `json:"mainModule"`
	MainVersion  string           `json:"mainVersion"`
	Dependencies []DependencyInfo `json:"dependencies"`
}

// GetBuildInfo extracts build information from the current binary
// This uses runtime/debug to get module information embedded at build time
func GetBuildInfo() *BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return &BuildInfo{
			GoVersion:    "unknown",
			MainModule:   "unknown",
			MainVersion:  "unknown",
			Dependencies: []DependencyInfo{},
		}
	}

	buildInfo := &BuildInfo{
		GoVersion:    info.GoVersion,
		MainModule:   info.Path,
		MainVersion:  info.Main.Version,
		Dependencies: make([]DependencyInfo, 0, len(info.Deps)),
	}

	// Extract dependencies
	for _, dep := range info.Deps {
		depInfo := DependencyInfo{
			Path:    dep.Path,
			Version: dep.Version,
		}
		if dep.Replace != nil {
			depInfo.Replace = dep.Replace.Path + "@" + dep.Replace.Version
		}
		buildInfo.Dependencies = append(buildInfo.Dependencies, depInfo)
	}

	// Sort dependencies by path for consistent output
	sort.Slice(buildInfo.Dependencies, func(i, j int) bool {
		return buildInfo.Dependencies[i].Path < buildInfo.Dependencies[j].Path
	})

	return buildInfo
}

// GetEVEVersion returns the version of the EVE module being used
// Returns "unknown" if EVE is not found in dependencies or running in dev mode
func GetEVEVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	// Check if this IS the EVE module
	if info.Path == "eve.evalgo.org" {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
		return "dev"
	}

	// Otherwise, look for EVE in dependencies
	for _, dep := range info.Deps {
		if dep.Path == "eve.evalgo.org" {
			if dep.Replace != nil {
				return dep.Replace.Version + " (replaced)"
			}
			return dep.Version
		}
	}

	return "unknown"
}

// GetDependency returns version information for a specific dependency
func GetDependency(modulePath string) *DependencyInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}

	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			depInfo := &DependencyInfo{
				Path:    dep.Path,
				Version: dep.Version,
			}
			if dep.Replace != nil {
				depInfo.Replace = dep.Replace.Path + "@" + dep.Replace.Version
			}
			return depInfo
		}
	}

	return nil
}
