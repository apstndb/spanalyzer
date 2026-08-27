//go:build integration

package main

import "github.com/apstndb/spanalyzer/cmd/spanner-query-gen/internal/runtimepins"

func querygenPinnedRuntimeImage(kind string) (string, error) {
	root, err := runtimepins.FindRepositoryRoot(".")
	if err != nil {
		return "", err
	}
	return runtimepins.ImageForHost(root, kind)
}
