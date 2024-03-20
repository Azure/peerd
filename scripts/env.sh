#!/bin/bash
set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $SCRIPT_DIR/env.az.sh

# Azure resources
RESOURCE_GROUP="p2p-ci-rg"
LOCATION="westus2"
AKS_NAME="acrp2pciaks"
ACR_NAME="acrp2pci"

indent() {
    sed 's/^/  /'
}

ensure_context_dir() {
    if [ "$DRY_RUN" == "false" ]; then
        ls "$CONTEXT_DIR" >/dev/null 2>&1 || mkdir $CONTEXT_DIR
    fi
}

print_and_exit_if_dry_run() {
    if [ "$DRY_RUN" == "true" ]; then
        echo
        echo
        echo "DRY RUN SUCCESSFUL: to confirm execution, re-run script with '-y'"
        exit 0
    fi
}

validate_prerequisites() {
    if ! get_prerequisites_versions ; then
        exit -1
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

validate_params() {
    local ec=2

    if [[ "$DRY_RUN" != "true" ]] && [[ "$DRY_RUN" != "false" ]]; then
        show_help
        echo "ERROR: dry run parameter invalid, expect true or false"
        exit $ec
    fi

    if [[ -z "$SUBSCRIPTION" ]]; then
        show_help
        echo "ERROR: subscription parameter is required"
        exit $ec
    fi

    if [[ -z "$RESOURCE_GROUP" ]]; then
        show_help
        echo "ERROR: resource group parameter is required"
        exit $ec
    fi

    if [[ -z "$LOCATION" ]]; then
        show_help
        echo "ERROR: location parameter is required"
        exit $ec
    fi
}

# Prepare local environment: try to install tools
get_prerequisites_versions() {
    local ec=1
    az --version >/dev/null 2>&1 || {
        echo "az cli not found: see https://learn.microsoft.com/en-us/cli/azure/install-azure-cli"
        return $ec
    }
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
    which uuid >/dev/null 2>&1 || {
        echo "uuid not found: to install, try 'apt-get install uuid'"
        return $ec
    }
}
