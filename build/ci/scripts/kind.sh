#!/bin/bash
set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
KIND_CLUSTER_NAME="p2p"
KIND_CLUSTER_CONTEXT="kind-$KIND_CLUSTER_NAME"
CLUSTER_CONFIG_FILE="$SCRIPT_DIR/../k8s/kind-cluster.yml"
HELM_CHART_DIR="$SCRIPT_DIR/../../package/peerd-helm"
HELM_RELEASE_NAME="peerd"
export GIT_ROOT="$(git rev-parse --show-toplevel)"

# Console colors and helpers
BG_BLUE="\e[44m"
GREEN="\e[32m"
YELLOW="\e[33m"
COLOR_RESET="\\033[0m"
NC='\033[0m' # No Color
RED='\e[01;31m'

echo_header() {
    echo -e "${BG_BLUE}=> $@ ${NC}"
}

indent() {
    sed 's/^/  /'
}

pipe_indent() {
    sed 's/^/│  /'
}

close_pipe_indent() {
    sed 's/^/└─ /'
}

print_and_exit_if_dry_run() {
    if [ "$DRY_RUN" == "true" ]; then
        echo
        echo
        echo "DRY RUN SUCCESSFUL: to confirm execution, re-run script with '-y'"
        exit 0
    fi
}

show_help() {
    usageStr="
Usage: $(basename $0) [OPTIONS]

This script is used for deploying apps to a local kind cluster for testing purposes.

Options:
  -h    Show help
  -y    Confirm execution, otherwise, it's a dry-run

Sub commands:
    cluster
        get
        create
        delete
    
    app
        deploy

* dry run: create new local environment
    $(basename $0) cluster create
* confirm: create new local environment
    $(basename $0) cluster create -y

* dry run: deploy app
    $(basename $0) app deploy
* confirm: deploy app
    $(basename $0) app deploy -y
"
    echo "$usageStr"
}

validate_prerequisites() {
    if ! get_prerequisites_versions ; then
        echo "You can also install prerequisites with:"
        echo
        echo make deps-install
        echo
        exit -1
    fi
}

# Validate tools available
get_prerequisites_versions() {
    local ec=1
    jq --version >/dev/null 2>&1 || {
        echo "jq not found: to install, try 'apt install jq'"
        return $ec
    }
    
    kubectl version --client=true >/dev/null 2>&1 || {
        echo "kubectl not found: see https://kubernetes.io/docs/tasks/tools/"
        return $ec
    }
    
    envsubst --version >/dev/null 2>&1 || {
        echo "envsubst not found: to install, try 'apt-get install gettext-base'"
        return $ec
    }
    
    kind --version >/dev/null 2>&1 || {
        echo "kind not found: see https://kind.sigs.k8s.io/docs/user/quick-start/#installing-from-release-binaries"
        return $ec
    }
}

validate_params() {
    local ec=2
    if [[ "$DRY_RUN" != "true" ]] && [[ "$DRY_RUN" != "false" ]]; then
        show_help
        echo "ERROR: dry run parameter invalid, expect true or false"
        exit $ec
    fi
}

print_p2p_metrics() {
    p=$(kubectl --context=$KIND_CLUSTER_CONTEXT -n peerd-ns get pods -l app=peerd -o jsonpath='{.items[*].metadata.name}')
    echo "pods: $p"

    for pod in $( echo "$p" | tr -s " " "\012" ); do
        echo "checking pod '$pod' for metrics"
        kubectl --context=$KIND_CLUSTER_CONTEXT -n peerd-ns exec -i $pod -- bash -c "cat /var/log/peerdmetrics"
        kubectl --context=$KIND_CLUSTER_CONTEXT -n peerd-ns exec -i $pod -- bash -c "curl http://localhost:5004/metrics/prometheus" | head -n 10
    done
}

show_cluster_info() {
    echo_header "Cluster Info for $KIND_CLUSTER_CONTEXT"
    kind get clusters | grep $KIND_CLUSTER_NAME 1>/dev/null
    if [ $? -ne 0 ]; then
        echo "Kind cluster not found"
    fi
    kubectl cluster-info --context=$KIND_CLUSTER_CONTEXT
    echo_header "Services"
    kubectl get services --all-namespaces
}

create_cluster() {
    echo_header "New cluster requested"
    if [ "$DRY_RUN" == "false" ]; then
        if [ $(kind get clusters | grep  $KIND_CLUSTER_NAME) ]; then
            echo "Cannot create cluster since it $KIND_CLUSTER_CONTEXT already exists"
            exit 1
        fi
        envsubst < $CLUSTER_CONFIG_FILE | kind create cluster --config -
        echo
    fi
}

delete_cluster() {
    echo_header "Deleting kind cluster: $KIND_CLUSTER_NAME"
    if [ "$DRY_RUN" == "false" ]; then
        kind delete cluster -n $KIND_CLUSTER_NAME
    fi
}

wait_for_events() {
    local context=$1
    local event=$2
    local minimumRequired=$3

    local ns="peerd-ns"
    local found=0

    # # Get app pods
    pods=$(kubectl --context=$context -n $ns get pods -o jsonpath='{.items[*].metadata.name}')
    echo "pods: $pods"
    total=`echo "$pods" | tr -s " " "\012" | wc -l`

    if [ -z "$minimumRequired" ]; then
        minimumRequired=$total
    fi

    # # Loop until all pods have the event or an error occurs.
    for ((i=1; i<=10; i++)); do
        found=0
        for pod in $( echo "$pods" | tr -s " " "\012" ); do
            echo "checking pod '$pod' for event '$event'"
            
            foundEvent=$(kubectl --context=$context -n $ns get events --field-selector involvedObject.kind=Pod,involvedObject.name=$pod -o json | jq -r ".items[] | select(.reason == \"$event\")")
            [[ "$foundEvent" == "" ]] && echo "Event '$event' not found for pod '$pod'" || found=$((found+1))
            
            errorEvent=$(kubectl --context=$context -n $ns get events --field-selector involvedObject.kind=Pod,involvedObject.name=$pod -o json | jq -r '.items[] | select(.reason == "P2PDisconnected" or .resosn == "P2PFailed")')
            [[ "$errorEvent" == "" ]] || (echo "Error event found for pod '$pod': $errorEvent" && exit 1)
        done

        if [ $found -eq $total ]; then
            echo "Success: All pods have event '$event'."
            break
        else
            echo "Waiting: $found out of $total pods have event '$event'. Attempt $i of 10."
            sleep 15
        fi
    done

    if [ $found -eq $total ]; then
        return
    elif [ $found -ge $minimumRequired ]; then
        echo "Warning: only $found out of $total pods have event '$event', but it meets the minimum criteria of $minimumRequired."
        return
    else
        echo "Validation failed"
        exit 1
    fi
}

cmd__test__random() {
    local img=$1

    echo "test image: $img"
    echo "initializing test 'random'"
    
    totalRequested=15
    ctr=0
    
    repos="oss/kubernetes/kubectl oss/kubernetes/kube-proxy oss/kubernetes/tiller oss/kubernetes/kube-api-server"

    echo "collecting $totalRequested secrets from mcr.microsoft.com repos: $repos"

    for repo in $repos; do
        if [ $ctr -ge $totalRequested ]; then
            break
        fi

        tags=$(curl https://mcr.microsoft.com/v2/$repo/tags/list 2> /dev/null | jq -r ".tags[]")
        
        for tag in $tags; do
            if [ $ctr -ge $totalRequested ]; then
                break
            fi
            
            manifest=`curl -s -H "Accept: application/vnd.oci.image.manifest.v1+json" -H "Accept: application/vnd.docker.distribution.manifest.v2+json" https://mcr.microsoft.com/v2/$repo/manifests/$tag`
            layers=`echo $manifest | jq -r ".layers[].digest"`

            for layer in $layers; do
                sas=`curl -v -s -H "Accept: application/vnd.oci.image.manifest.v1+json" -H "Accept: application/vnd.docker.distribution.manifest.v2+json" https://mcr.microsoft.com/v2/$repo/blobs/$layer 2>&1 | grep "< location: " | awk -F ': ' '{print $2}'`
                secret="$secret $sas"
                ctr=$((ctr+1))
            done

        done

    done

    if [ "$DRY_RUN" == "true" ]; then
        echo "[dry run] collected $ctr secrets"
    else
        echo "collected $ctr secrets"
    fi

    echo "loading test image and deploying"
    if [ "$DRY_RUN" == "false" ]; then
        kind load docker-image $img --name $KIND_CLUSTER_NAME
        export TEST_RANDOM_CONTAINER_IMAGE=$img
        export SECRETS=$secret
        export NODE_COUNT=3
        envsubst < $SCRIPT_DIR/../k8s/test-random.yml | kubectl --context=$KIND_CLUSTER_CONTEXT apply -f -

        echo "waiting for logs" && \
            sleep 10 && \
            kubectl --context=$KIND_CLUSTER_CONTEXT -n peerd-ns logs -l app=random -f
    
        kubectl --context=$KIND_CLUSTER_CONTEXT -n peerd-ns delete ds/random

        echo "checking p2p active pods" && sleep 2
        wait_for_events $KIND_CLUSTER_CONTEXT "P2PActive" 1
    fi

    echo "fetching metrics from pods"
    print_p2p_metrics
}

cmd__cluster__get() {
    show_cluster_info
}

cmd__cluster__create() {
    create_cluster
    print_and_exit_if_dry_run
    echo
    echo "Hooray! Local cluster is available  🙌🐳"
    echo
    exit 0
}

cmd__cluster__delete() {
    delete_cluster
}

cmd__app__deploy() {
    export PEERD_CONTAINER_IMAGE_REF=$1

    kubectl cluster-info --context=$KIND_CLUSTER_CONTEXT && \
        echo_header "Deploying app: $PEERD_CONTAINER_IMAGE_REF"
    
    if [ "$DRY_RUN" == "false" ]; then
        echo "loading image"
        kind load docker-image $PEERD_CONTAINER_IMAGE_REF --name $KIND_CLUSTER_NAME
        
        helm --kube-context=$KIND_CLUSTER_CONTEXT status $HELM_RELEASE_NAME >/dev/null 2>&1 && \
            helm --kube-context=$KIND_CLUSTER_CONTEXT uninstall $HELM_RELEASE_NAME

        echo "Helm installing release:" $HELM_RELEASE_NAME
        helm --kube-context=$KIND_CLUSTER_CONTEXT install --wait $HELM_RELEASE_NAME $HELM_CHART_DIR \
            --set peerd.image.ref=$PEERD_CONTAINER_IMAGE_REF

        echo "waiting for pods to connect"
        wait_for_events $KIND_CLUSTER_CONTEXT "P2PConnected" 3
    fi
}

get_opts() {
    while getopts 'yh' OPTION; do
        case "$OPTION" in
            y)
                DRY_RUN="false"
                ;;
            h)
                show_help
                exit 1 # exit non-zero to break invocation of command
                ;;
        esac
    done
    shift $((OPTIND-1))
}

# Initialize script and validate prerequisites.
if [[ -z "$DRY_RUN" ]]; then
    DRY_RUN="true"
fi

validate_params
validate_prerequisites

# Check sub command then check fall through to
# main command if sub command doesn't exist
# functions that are entry points should be of the form
# cmd__{command}__{subcommand} or cmd__{command}
if declare -f "cmd__${1}__${2}" >/dev/null; then
    func="cmd__${1}__${2}"
    shift; shift;
    get_opts $@
    # pop $1 $2 off the argument list
    "$func" "$2"    # invoke our named function w/ all remaining arguments
elif declare -f "cmd__$1" >/dev/null; then
    func="cmd__$1"
    shift; # pop $1 off the argument list
    get_opts $@
    "$func" "$@"    # invoke our named function w/ all remaining arguments
else
    echo "Neither command $1 nor subcommand ${1} ${2} recognized" >&2
    show_help
    exit 1
fi
