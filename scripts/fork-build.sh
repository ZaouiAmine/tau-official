#!/usr/bin/env bash
# Fork build: scripts/fork-build.sh [version] -> bin/{tau.exe,tau,dream,tau-node} stamped with Version/Commit/BuildDate (default: git describe v*-fork.*).
set -euo pipefail
cd "$(dirname "$0")/.."
V=${1:-$(git describe --tags --match 'v*-fork.*')}
C=$(git rev-parse --short HEAD)
D=$(date -u +%Y-%m-%dT%H:%M:%SZ)
M=github.com/taubyte/tau
export CGO_ENABLED=${CGO_ENABLED:-0}
b() { # goos out pkg ldflags
	GOOS=$1 GOARCH=amd64 go build -trimpath -ldflags "-s -w $4" -o "bin/$2" "$3"
}
TAU="-X $M/tools/tau/common.LinkVersion=$V -X $M/tools/tau/common.LinkCommit=$C"
b windows tau.exe ./tools/tau "$TAU"
b linux tau ./tools/tau "$TAU"
b linux dream ./tools/dream "-X $M/dream.LinkVersion=$V -X $M/dream.LinkCommit=$C"
b linux tau-node . "-X $M/cli/app.Version=$V -X $M/cli/app.Commit=$C -X $M/cli/app.BuildDate=$D"
echo "built $V ($C)"
