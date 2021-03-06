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

export GOPATH="${WORKSPACE:-.}/git"
mkdir -p ${GOPATH}


cd "${WORKSPACE}/git/src"

# need to clone our other deps

#git clone git@scm-main-01.dc.myfitnesspal.com:Metrics/consistent.git consistent
#git clone git@scm-main-01.dc.myfitnesspal.com:Metrics/statsd.git statsd

cd "${WORKSPACE}/git"

# grab the external pacakges

progress Grabing external packages

#go get github.com/BurntSushi/toml
#go get github.com/davecheney/profile
#go get github.com/reusee/mmh3
#go get gopkg.in/op/go-logging.v1
#go get github.com/smartystreets/goconvey/convey
#go get github.com/go-sql-driver/mysql
#go get github.com/gocql/gocql
#go get github.com/robyoung/go-whisper
#go get github.com/jbenet/go-reuseport
#go get github.com/Shopify/sarama

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

cp -rf cadent ${OUTPUT}
cp -rf echoserver ${OUTPUT}
cp -rf statblast ${OUTPUT}

#tar -cvzf "${OUTPUT}/${APP_NAME}-${BUILDID}.tmp" -C ${PACKAGE_BASE}


### make changelog for the debian maker
progress Creating changelog

GIT_VERSION=$(git log -1 | head -1 | cut -d " " -f2 | cut -c 1-7)
ONDATE=$(date +"%a, %d %b %Y %T %z")
ON_VERIONS=$(cat ./version)
cat > pkgs/debian/changelog <<EOF
mfp-cadent (${ON_VERIONS}.${GIT_VERSION}) unstable; urgency=low

  * git head (${ON_VERIONS}.${GIT_VERSION})

 -- Bo Blanton <boblanton@myfitnesspal.com>  ${ONDATE}

EOF

progress Build complete.



exit 0
