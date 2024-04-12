#!/bin/bash
set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $SCRIPT_DIR/env.sh

PEERD_HELM_CHART="$SCRIPT_DIR/../../package/peerd-helm"
TELEPORT_DEPLOY_TEMPLATE="$SCRIPT_DIR/../k8s/teleport.yml"
SCANNER_APP_DEPLOY_TEMPLATE="$SCRIPT_DIR/../k8s/scanner.yml"
TESTS_AZURE_CLI_DEPLOY_TEMPLATE=$SCRIPT_DIR/../k8s/azure-cli.yml

show_help() {
    usageStr="
Usage: $(basename $0) [OPTIONS]

This script is used for deploying apps to an AKS cluster for testing purposes.

Options:
  -h    Show help
  -y    Confirm execution, otherwise, it's a dry-run

Sub commands:
    nodepool
        up
        delete

    init
        random

    run
        dotnet

* dry run: create nodepool called 'nodepool1' and install the peerd proxy
    $(basename $0) nodepool up nodepool1

* confirm: create nodepool called 'nodepool1' and install the peerd proxy
    $(basename $0) nodepool up -y nodepool1

* dry run: delete nodepool 'nodepool1'
    $(basename $0) delete nodepool 'nodepool1'

* confirm: delete nodepool 'nodepool1'
    $(basename $0) nodepool delete -y 'nodepool1'

* dry run: runs the ctr test on 'nodepool1'
    $(basename $0) test ctr 'nodepool1'

* confirm: run the ctr test on 'nodepool1'
    $(basename $0) test ctr -y 'nodepool1'

* dry run: runs the streaming test on 'nodepool1'
    $(basename $0) test streaming 'nodepool1'

* confirm: run the streaming test on 'nodepool1'
    $(basename $0) test streaming -y 'nodepool1'
"
    echo "$usageStr"
}

get_aks_credentials() {
    local cluster=$1
    local rg=$2

    az aks get-credentials --resource-group $rg --name $cluster --overwrite-existing && \
        kubelogin convert-kubeconfig -l azurecli && \
        kubectl cluster-info
}

nodepool_deploy() {
    local aksName=$1
    local rg=$2
    local nodepool=$3

    if [ "$DRY_RUN" == "false" ]; then
        echo "creating nodepool '$nodepool' in aks cluster '$aksName' in resource group '$rg'" && \
            az aks nodepool add --cluster-name $aksName --name $nodepool --resource-group $rg \
                --mode User --labels "p2p-nodepool=$nodepool" --node-count 3 --node-vm-size Standard_D2s_v3
    else
        echo "[dry run] would have deployed nodepool '$nodepool' to aks cluster '$aksName' in resource group '$rg'"
    fi

}

peerd_helm_deploy() {
    local nodepool=$1
    local peerd_image_tag=$2
    local configureMirrors=$3

    ensure_azure_token
    
    echo "deploying peerd to k8s cluster, chart: '$PEERD_HELM_CHART', tag: '$peerd_image_tag'" && \
        kubectl cluster-info

    if [ "$DRY_RUN" == "false" ]; then
        HELM_RELEASE_NAME=peerd && \
            helm install --wait $HELM_RELEASE_NAME $PEERD_HELM_CHART \
                --set "peerd.image.ref=ghcr.io/azure/acr/dev/peerd:$peerd_image_tag" \
                --set "peerd.configureMirrors=$configureMirrors"
    else
        echo "[dry run] would have deployed app to k8s cluster"
    fi

    print_and_exit_if_dry_run
}

wait_for_peerd_pods() {
    local cluster=$1
    local rg=$2
    local nodepool=$3
    local event=$4
    local minimumRequired=$5

    local found=0

    # Get the list of pods.
    pods=$(kubectl -n peerd-ns get pods -l app=peerd -o jsonpath='{.items[*].metadata.name}')
    echo "pods: $pods"
    total=`echo "$pods" | tr -s " " "\012" | wc -l`

    if [ -z "$minimumRequired" ]; then
        minimumRequired=$total
    fi

    # Loop until all pods are connected or an error occurs.
    for ((i=1; i<=10; i++)); do
        # Initialize a counter for connected pods.
        found=0

        # Loop through each pod.
        for pod in $( echo "$pods" | tr -s " " "\012" ); do
            echo "checking pod '$pod' for event '$event'"
            
            foundEvent=$(kubectl -n peerd-ns get events --field-selector involvedObject.kind=Pod,involvedObject.name=$pod -o json | jq -r ".items[] | select(.reason == \"$event\")")
            [[ "$foundEvent" == "" ]] && echo "Event '$event' not found for pod '$pod'" || found=$((found+1))
            
            errorEvent=$(kubectl -n peerd-ns get events --field-selector involvedObject.kind=Pod,involvedObject.name=$pod -o json | jq -r '.items[] | select(.reason == "P2PDisconnected" or .resosn == "P2PFailed")')
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

print_peerd_metrics() {
    p=$(kubectl -n peerd-ns get pods -l app=peerd -o jsonpath='{.items[*].metadata.name}')
    echo "pods: $p"

    for pod in $( echo "$p" | tr -s " " "\012" ); do
        echo "checking pod '$pod' for metrics"
        kubectl -n peerd-ns exec -i $pod -- bash -c "cat /var/log/peerdmetrics"
        kubectl --context=$KIND_CLUSTER_CONTEXT -n peerd-ns exec -i $pod -- bash -c "curl http://localhost:5004/metrics/prometheus" | head -n 20 | echo " ..."
    done
}

cmd__nodepool__delete() {
    local aksName=$AKS_NAME
    local rg=$RESOURCE_GROUP
    local nodepool=$1

    if [ "$DRY_RUN" == "false" ]; then
        echo "deleting nodepool '$nodepool' in aks cluster '$aksName' in resource group '$rg'" && \
            az aks nodepool delete --cluster-name $aksName --name $nodepool --resource-group $rg
    else
        echo "[dry run] would have deleted nodepool '$nodepool' in aks cluster '$aksName' in resource group '$rg'"
    fi
}

cmd__nodepool__up () {
    local nodepool=$1
    local peerd_image_tag=$PEERD_IMAGE_TAG
    local configureMirrors=$PEERD_CONFIGURE_MIRRORS

    echo "get AKS credentials"
    get_aks_credentials $AKS_NAME $RESOURCE_GROUP

    echo "sanitizing"
    helm uninstall peerd --ignore-not-found=true

    echo "creating new nodepool '$nodepool'"
    nodepool_deploy $AKS_NAME $RESOURCE_GROUP $nodepool

    echo "deploying peerd helm chart using tag '$peerd_image_tag'"
    peerd_helm_deploy $nodepool $peerd_image_tag $configureMirrors

    echo "waiting for pods to connect"
    wait_for_peerd_pods $AKS_NAME $RESOURCE_GROUP $nodepool "P2PConnected"
}

cmd__test__ctr() {
    aksName=$AKS_NAME
    rg=$RESOURCE_GROUP
    local nodepool=$1

    echo "running test 'ctr'"

    if [ "$DRY_RUN" == "true" ]; then
        echo "[dry run] would have run test 'ctr'"
    else
        # Pull the image on all nodes and verify that at least one P2PActive event is generated.
        kubectl apply -f $TESTS_AZURE_CLI_DEPLOY_TEMPLATE

        wait_for_peerd_pods $context $AKS_NAME $RESOURCE_GROUP $nodepool "P2PActive" 1

        echo "fetching metrics from pods"
        print_peerd_metrics

        echo "cleaning up apps"
        helm uninstall peerd --ignore-not-found=true
        kubectl delete -f $TESTS_AZURE_CLI_DEPLOY_TEMPLATE

        echo "test 'ctr' complete"
    fi

    print_and_exit_if_dry_run
}

cmd__test__streaming() {
    aksName=$AKS_NAME
    rg=$RESOURCE_GROUP
    local nodepool=$1

    echo "running test 'streaming'"

    if [ "$DRY_RUN" == "true" ]; then
        echo "[dry run] would have run test 'streaming'"
    else
        echo "deploying acr mirror"
        kubectl apply -f $TELEPORT_DEPLOY_TEMPLATE

        echo "waiting 5 minutes" 
        sleep 300

        echo "deploying scanner app and waiting 2 minutes"
        envsubst < $SCANNER_APP_DEPLOY_TEMPLATE | kubectl apply -f -
        sleep 120

        echo "scanner logs"
        kubectl -n peerd-ns logs -l app=tests-scanner

        wait_for_peerd_pods $context $AKS_NAME $RESOURCE_GROUP $nodepool "P2PActive" 1

        echo "fetching metrics from pods"
        print_peerd_metrics

        echo "cleaning up apps"
        helm uninstall peerd --ignore-not-found=true
        kubectl delete -f $SCANNER_APP_DEPLOY_TEMPLATE

        echo "test 'streaming' complete"
    fi

    print_and_exit_if_dry_run
}

# Initialize script.
if [[ -z "$DRY_RUN" ]]; then
    DRY_RUN="true"
fi

validate_params
validate_prerequisites

echo $@

# Check sub command then check fall through to
# main command if sub command doesn't exist
# functions that are entry points should be of the form
# cmd__{command}__{subcommand} or cmd__{command}
if declare -f "cmd__${1}__${2}" >/dev/null; then
    func="cmd__${1}__${2}"
    
    # pop $1 $2 off the argument list
    shift; shift;

    get_opts $@
    
    "$func" "$2"    # invoke our named function w/ all remaining arguments
elif declare -f "cmd__$1" >/dev/null; then
    func="cmd__$1"
    shift; # pop $1 off the argument list
    get_opts $@
    "$func" "$1"    # invoke our named function w/ all remaining arguments
else
    echo "Neither command $1 nor subcommand ${1} ${2} recognized" >&2
    show_help
    exit 1
fi
