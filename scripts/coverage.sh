#!/bin/bash
set -e

## Parameters
source_dir=${1?Source directory is required. Please pass as a parameter the absolute path to the directory of the source code to be tested.}
test_pkgs=${2?Test packages are required. Please pass as a parameter the packages to test.}
clear_COVERAGE_DIR=${3:-false}
test_params=${4:-"-timeout 240s"}

## Variables
script_dir="$( dirname "${BASH_SOURCE[0]}" )"
initial_dir=$( pwd )

## Variables depending on sources
results_dir="$dest_dir/$COVERAGE_DIR"

## Functions
show_help() {
    usageStr="
This script creates coverage for golang projects.
As a part of this, it does the following tasks.
    - Installs the following support modules if they are not already installed:
        - gotestsum
        - gcov2lcov
        - gocov
        - gocov-xml
    - Runs the tests
        - This creates the following files:
            - coverage.txt
            - jsongotest.log
            - coverage.xml
    - Generates the following test results (if the coverage.txt file was generated):
        - coverage/coverage.html
        - coverage/coverage.txt
        - coverage/lcov.info
        - coverage/coverage.cobertura.xml

Parameters:
    Source Directory              (required)              The absolute path to the directory of the source code to be tested
    Test Packages                 (required)              The list of packages to test
    Clear Test Results Directory  (default: false)        Controls whether or not any existing Test Results Directory is cleared
    Test Parameters               (default: timeout 240s) Additional test parameters to pass to the test wrapper

EXAMPLES:
    coverage.sh /path/to/go/project 'package1 package2 package3 package4' true
        - Executes tests related to packages 1-4 in the folder '/path/to/go/project' and clears the test results directory if it exists
    coverage.sh /path/to/go/project 'packageA packageB' false '-timeout 5s'
        - Executes tests related to packages A and B in the folder '/path/to/go/project', does not clear any existing test results directory, and passes the timeout parameter to the testing wrapper, with a timeout of 5 seconds.
"
    echo "$usageStr"
}

## Main
echo -e "\n------ Generating test results ------\n"

echo "Current working directory: $initial_dir"
echo "Script directory: $script_dir"
echo -e "Source directory: $source_dir\n"

# If any of the required modules are not installed, notify the user to install them
if [ -z $(command -v "gotestsum") ] || [ -z $(command -v "gotestsum") ] || [ -z $(command -v "gotestsum") ] || [ -z $(command -v "gotestsum") ]; then

    echo -e "\nPlease install the required modules and run this script again."
    echo -e "The script to install the missing modules can be found at $script_dir/install-go-modules.sh"
    exit 1

fi;

cd $source_dir

if [[ $clear_COVERAGE_DIR = true ]] && [ -d "$COVERAGE_DIR" ]; then
    echo -e "\nClearing test results directory\n"
    rm -rf $COVERAGE_DIR
fi;

if [ ! -d "$COVERAGE_DIR" ]; then
    echo -e "\nCreating test results directory\n"
    mkdir -p $COVERAGE_DIR
fi;

echo -e "\n------ Running tests ------\n"
## coverage.txt format - https://github.com/golang/go/blob/0104a31b8fbcbe52728a08867b26415d282c35d2/src/cmd/cover/profile.go#L56
gotestsum --format standard-verbose --junitfile $COVERAGE_DIR/coverage.xml --jsonfile $COVERAGE_DIR/jsongotest.log -- -cover -coverprofile=$COVERAGE_DIR/coverage.txt -covermode=atomic $test_params $test_pkgs | tee $COVERAGE_DIR/testoutput.txt

echo -e "\n------ Generating coverage - lcov ------\n"
GOROOT=$(go env GOROOT) gcov2lcov -infile=$COVERAGE_DIR/coverage.txt -outfile=$COVERAGE_DIR/lcov.info

echo -e "------ Generating coverage - cobertura ------\n"
GOROOT=$(go env GOROOT) gocov convert $COVERAGE_DIR/coverage.txt | gocov-xml > $COVERAGE_DIR/coverage.cobertura.xml

echo -e "------ Generating coverage - html ------\n"
go tool cover -html=$COVERAGE_DIR/coverage.txt -o $COVERAGE_DIR/coverage.html

cd $initial_dir