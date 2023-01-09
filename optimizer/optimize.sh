#!/bin/ash
# shellcheck shell=dash
# See https://www.shellcheck.net/wiki/SC2187
set -o errexit -o nounset -o pipefail
command -v shellcheck >/dev/null && shellcheck "$0"

export PATH=$PATH:/root/.cargo/bin

# Suffix for non-Intel built artifacts
MACHINE=$(uname -m)
SUFFIX=${MACHINE#x86_64}
SUFFIX=${SUFFIX:+-$SUFFIX}

echo "Info: RUSTC_WRAPPER=$RUSTC_WRAPPER"

echo "Info: sccache stats before build"
sccache -s

mkdir -p artifacts
rm -f artifacts/checksums_intermediate.txt

CRATE="$1"
RUSTFLAGS='-C link-arg=-s' cargo build -p "$CRATE" --release --lib --target wasm32-unknown-unknown --locked
WASM="$(echo "$CRATE" | tr - _)"
WASM="/code/target/wasm32-unknown-unknown/release/$WASM.wasm"
NAME=$(basename "$WASM" .wasm)${SUFFIX}.wasm
echo "Creating intermediate hash for $NAME ..."
sha256sum -- "$WASM" | tee -a artifacts/checksums_intermediate.txt
echo "Optimizing $NAME ..."
wasm-opt -Os "$WASM" -o "artifacts/$NAME"

# create hash
echo "Creating hashes ..."
(
  cd artifacts
  sha256sum -- *.wasm | tee checksums.txt
)

echo "Info: sccache stats after build"
sccache -s

echo "done"