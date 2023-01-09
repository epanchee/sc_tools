package builder

import (
	"os"
)

const (
	containerName             = "cosmwasm-builder"
	patchedRustOptimizerImage = "fixed_optimizer:local"
)

type WasmBuilder interface {
	BuildWasm(repoDir, projectName, crateName string, allowArm bool) ([]byte, error)
}

func readWasmFile(path string) []byte {
	wasmFile, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	} else {
		return wasmFile
	}
}
