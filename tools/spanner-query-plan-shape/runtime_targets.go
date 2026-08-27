package main

import "github.com/apstndb/spanalyzer/tools/internal/runtimepins"

func repositoryPinnedImage(kind string) (string, error) {
	root, err := runtimepins.FindRepositoryRoot(".")
	if err != nil {
		return "", err
	}
	return runtimepins.ImageForHost(root, kind)
}
