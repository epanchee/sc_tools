package cmd

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sc-tools/builder"
	"sc-tools/repo"
)

func checkoutAndBuild(flags map[string]string, allowArm, patched bool) ([]byte, error) {
	link := flags["commit-link"]
	crateName := flags["crate-name"]

	repoInfo, err := repo.ParseRepoLink(link)
	if err != nil {
		return nil, err
	}

	repoDir := repo.GenTempDirPath()
	log.Info("Cloning repo ...")
	cleanup, err := repo.FetchRepo(&repoInfo, repoDir)
	defer cleanup()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch repo")
	}

	wasmBuilder := func() builder.WasmBuilder {
		if patched {
			log.Info("Building wasm using patched optimizer image")
			return builder.PatchedBuilder{}
		} else {
			log.Info("Building wasm using ", flags["image"])
			return builder.CosmWasmBuilder{BuilderImage: flags["image"]}
		}
	}()
	wasmByteCode, err := wasmBuilder.BuildWasm(repoDir, repoInfo.ProjectName, crateName, allowArm)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build wasm")
	}

	return wasmByteCode, nil
}
