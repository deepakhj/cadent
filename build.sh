#!/bin/bash
## this script is actually being sourced from jenkins

declare -i START=$(date +%s)
declare -i CHECKPOINT=$(date +%s)

function progress(){
        local NOW=$(date +%s)
        local elapsed_total=$(( NOW - START ))
        local elapsed=$(( NOW - CHECKPOINT ))
        echo "Progress: ${elapsed} secs (${elapsed_total} secs):" "${@}"
        CHECKPOINT=${NOW}
}

export WORKSPACE="${WORKSPACE:-.}"
export SOURCE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"

cd "${WORKSPACE}/git"

make clean

make

progress Staging for angstrom packager
"${SOURCE}/angstrom/package" "${APP_NAME:-unknown}" "${BUILDID:-develop}" "${WORKSPACE:-.}/staging" "${WORKSPACE:-.}/output"
progress Build complete.