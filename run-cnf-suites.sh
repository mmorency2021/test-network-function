#!/usr/bin/env bash

# defaults
export OUTPUT_LOC="$PWD/test-network-function"

usage() {
	echo "$0 [-o OUTPUT_LOC] [-f SUITE...] -s [SUITE...]"
	echo "Call the script and list the test suites to run"
	echo "  e.g."
	echo "    $0 [ARGS] -f access-control lifecycle"
	echo "  will run the access-control and lifecycle suites"
	echo ""
	echo "Allowed suites are listed in the README."
}

usage_error() {
	usage
	exit 1
}

FOCUS=""
SKIP=""
# Parge args beginning with "-"
while [[ $1 == -* ]]; do
	case "$1" in
		-h|--help|-\?) usage; exit 0;;
		-o) if (($# > 1)); then
				  OUTPUT_LOC=$2; shift
			  else
				  echo "-o requires an argument" 1>&2
				  exit 1
			  fi ;;
    -s|--skip)
        while (( "$#" >= 2 )) && ! [[ $2 = --* ]] && ! [[ $2 = -* ]] ; do
          SKIP="$2|$SKIP"
          shift
        done;;
		-f|--focus)
        while (( "$#" >= 2 )) && ! [[ $2 = --* ]]  && ! [[ $2 = -* ]] ; do
          FOCUS="$2|$FOCUS"
          shift
        done;;
    -*) echo "invalid option: $1" 1>&2; usage_error;;
	esac
  shift
done
# specify Junit report file name.
GINKGO_ARGS="-junit $OUTPUT_LOC -claimloc $OUTPUT_LOC -ginkgo.reportFile $OUTPUT_LOC/cnf-certification-tests_junit.xml -ginkgo.v -test.v"


# If no focus is set then display usage and quit with a non-zero exit code.
[ -z "$FOCUS" ] && echo "no focus found" && usage_error

FOCUS=${FOCUS%?}  # strip the trailing "|" from the concatenation
SKIP=${SKIP%?} # strip the trailing "|" from the concatenation

# Run cnf-feature-deploy test container if not running inside a container
# cgroup file doesn't exist on MacOS. Consider that as not running in container as well
if [[ ! -f "/proc/1/cgroup" ]] || grep -q init\.scope /proc/1/cgroup; then
	cd script
	./run-cfd-container.sh
	cd ..
fi

if [[ -z "${TNF_PARTNER_SRC_DIR}" ]]; then
	echo "env var \"TNF_PARTNER_SRC_DIR\" not set, running the script without updating infra"
else
	make -C $TNF_PARTNER_SRC_DIR install-partner-pods
fi

echo "Running with focus '$FOCUS'"
echo "Running with skip  '$SKIP'"
echo "Report will be output to '$OUTPUT_LOC'"
echo "ginkgo arguments '${GINKGO_ARGS}'"
cd ./test-network-function && ./test-network-function.test -ginkgo.focus="$FOCUS" -ginkgo.skip="$SKIP" ${GINKGO_ARGS}
