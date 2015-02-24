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

export GOPATH="${WORKSPACE:-.}/gopath"
mkdir -p ${GOPATH}


cd "${WORKSPACE}/git"

# need to clone our other deps

git clone git@scm-main-01.dc.myfitnesspal.com:goutil/consistent.git cmd/consthash/consistent
git clone git@scm-main-01.dc.myfitnesspal.com:goutil/statsd.git cmd/consthash/statsd

# grab the external pacakges

progress Grabing external packages

go get github.com/bbangert/toml
go get github.com/davecheney/profile

make clean

progress Building .... 

make

progress Copying artifacts

export TARGET="${WORKSPACE:-.}/staging"
export OUTPUT="${WORKSPACE:-.}/output"


## clean existing staging output
echo Clean existing staging directory
rm -rf ${TARGET}
echo Clean existing output directory
rm -rf ${OUTPUT}

progress Copying artifacts
mkdir -p ${TARGET}
mkdir -p ${OUTPUT}

cp -rf html ${OUTPUT}
cp -rf consthash ${OUTPUT}
cp -rf echoserver ${OUTPUT}

#tar -cvzf "${OUTPUT}/${APP_NAME}-${BUILDID}.tmp" -C ${PACKAGE_BASE}


### make changelog for the debian maker
progress Creating changelog

GIT_VERSION=$(git log -1 | head -1 | cut -d " " -f2 | cut -c 1-7)
ONDATE=$(date +"%a, %d %b %Y %T %z")
ON_VERIONS=$(cat ./version)
cat > pkgs/debian/changelog <<EOF
mfp-consthash (${ON_VERIONS}.${GIT_VERSION}) unstable; urgency=low

  * git head

 -- Bo Blanton <boblanton@myfitnesspal.com>  ${ONDATE}

EOF

## remove lock file if no other proc is running
RUNNING=`ps --no-headers -Creprepro | wc -l`
if [ ${RUNNING} -gt 1 ]; then
  echo "Previous reprepro is still running. must exit"
  exit 1
fi 

sudo rm -f /vol/mfp-jenkins-deb/repo//db/lockfile

#progress Staging for angstrom packager
#"${SOURCE}/angstrom/package" "${APP_NAME:-unknown}" "${BUILDID:-develop}" "${WORKSPACE:-.}/staging" "${WORKSPACE:-.}/output"
progress Build complete.



exit 0
