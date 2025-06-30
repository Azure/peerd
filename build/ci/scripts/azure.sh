#!/bin/bash
set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $SCRIPT_DIR/env.sh

PEERD_HELM_CHART="$SCRIPT_DIR/../../package/peerd-helm"
SCANNER_APP_DEPLOY_TEMPLATE="$SCRIPT_DIR/../k8s/scanner.yml"
PEERD_LLM_CI_DEPLOY_TEMPLATE=$SCRIPT_DIR/../k8s/llm.yml

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

* dry run: create nodepool called 'nodepool1' and install the peerd proxy
    $(basename $0) nodepool up nodepool1

* confirm: create nodepool called 'nodepool1' and install the peerd proxy
    $(basename $0) nodepool up -y nodepool1

* dry run: delete nodepool 'nodepool1'
    $(basename $0) delete nodepool 'nodepool1'

* confirm: delete nodepool 'nodepool1'
    $(basename $0) nodepool delete -y 'nodepool1'

* dry run: runs the llm streaming test on 'nodepool1'
    $(basename $0) test llm_streaming 'nodepool1'

* confirm: run the llm streaming test on 'nodepool1'
    $(basename $0) test llm_streaming -y 'nodepool1'
"
    echo "$usageStr"
}

trap 'on_error' ERR

on_error() {
    echo "[on_error] script encountered an error, cleaning up"
    cmd__peerd_pod_watcher__stop || echo "[on_error] failed to stop peerd pod watcher"
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
    local configureOverlaybdP2p=$4

    azCmd="az aks nodepool add --cluster-name $aksName --name $nodepool --resource-group $rg \
                --mode User --node-count 1 --labels "peerd=ci" --node-vm-size $NODE_VM_SIZE"
    if [ "$configureOverlaybdP2p" == "true" ]; then
        azCmd="$azCmd --enable-artifact-streaming"
    fi

    if [ "$DRY_RUN" == "false" ]; then
        echo "creating nodepool '$nodepool' in aks cluster '$aksName' in resource group '$rg'" && $azCmd

        if [ "$configureOverlaybdP2p" == "true" ]; then
            helm install --wait overlaybd-p2p $SCRIPT_DIR/../../../tools/configure-overlaybd-p2p-helm \
                --set "overlaybd.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key=peerd" \
                --set "overlaybd.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator=In" \
                --set "overlaybd.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0]=ci"
        fi
    else
        echo "[dry run] would have run command: '$azCmd'"
        echo "[dry run] would have deployed overlaybd p2p configuration helm chart"
    fi
}

peerd_helm_deploy() {
    local nodepool=$1
    local peerd_image_tag=$2
    local configureMirrors=$3
    
    echo "deploying peerd to k8s cluster, chart: '$PEERD_HELM_CHART', tag: '$peerd_image_tag'" && \
        kubectl cluster-info

    if [ "$DRY_RUN" == "false" ]; then
        HELM_RELEASE_NAME=peerd && \
            helm install --wait $HELM_RELEASE_NAME $PEERD_HELM_CHART \
                --set "peerd.image.ref=ghcr.io/azure/acr/dev/peerd:$peerd_image_tag" \
                --set "peerd.resources.limits.memory=4Gi" \
                --set "peerd.resources.limits.cpu=2" \
                --set "peerd.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key=peerd" \
                --set "peerd.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator=In" \
                --set "peerd.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0]=ci"
    else
        echo "[dry run] would have deployed app to k8s cluster"
    fi

    print_and_exit_if_dry_run
}

wait_for_pod_events() {
    local cluster=$1
    local rg=$2
    local nodepool=$3
    local event=$4
    local selector=$5
    local minimumRequired=$6

    local found=0

    # Get the list of pods.
    pods=$(kubectl -n peerd-ns get pods -l $selector -o jsonpath='{.items[*].metadata.name}')
    echo "pods: $pods"
    total=`echo "$pods" | tr -s " " "\012" | wc -l`

    if [ -z "$minimumRequired" ]; then
        minimumRequired=$total
    fi

    # Loop until all pods have the event.
    for ((i=1; i<=300; i++)); do
        # Initialize a counter for successfully checked pods.
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
            return
        elif [ $found -ge $minimumRequired ]; then
            echo "$found out of $total pods have event '$event', which meets the minimum criteria of $minimumRequired."
            return
        else
            echo "Waiting: $found out of $total pods have event '$event'. Attempt $i of 300."
            sleep 15
        fi
    done

    echo "Validation failed."
    exit 1
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

peerd_pod_watcher() {
    echo "[Pod Watcher] Starting peerd pod watcher"
    local event="P2PConnected"

    for ((i = 1; i <= 300; i++)); do
        # Get the list of pods with the label `app=peerd`
        pods=$(kubectl -n peerd-ns get pods -l app=peerd -o jsonpath='{.items[*].metadata.name}')
        if [ -z "$pods" ]; then
            echo "[Pod Watcher] No pods found with label 'app=peerd'. Retrying..."
            sleep 15
            continue
        fi

        # echo "[Pod Watcher] Found peerd pods: $pods"

        # Check each pod for the specified event
        for pod in $pods; do
            # echo "[Pod Watcher] Checking pod '$pod' for event '$event'"

            if kubectl -n peerd-ns get events --field-selector involvedObject.kind=Pod,involvedObject.name="$pod" -o json | \
                jq -e ".items[] | select(.reason == \"$event\")" > /dev/null; then
                # echo "[Pod Watcher] Event '$event' found for pod '$pod'"

                NODE_NAME=$(kubectl -n peerd-ns get pod "$pod" -o jsonpath='{.spec.nodeName}')
                if [ -n "$NODE_NAME" ]; then
                    currentLabel=$(kubectl get node $NODE_NAME -o jsonpath='{.metadata.labels.peerd-status}')
                    if [ "$currentLabel" == "connected" ]; then
                        # echo "[Pod Watcher] Node '$NODE_NAME' already labeled with 'peerd-status=connected'"
                        continue
                    else
                        sleep 5
                        echo "[Pod Watcher] Labeling node '$NODE_NAME' with 'peerd-status=connected'"
                        kubectl label node "$NODE_NAME" peerd-status=connected --overwrite
                    fi
                else
                    echo "[Pod Watcher] Failed to get node name for pod '$pod'"
                fi
            else
                echo "[Pod Watcher] Event '$event' not found for pod '$pod'"
            fi
        done

        sleep 15
    done

    echo "[Pod Watcher] Finished watching pods after 300 iterations."
}

cmd__peerd_pod_watcher__start() {
    # Start the peerd pod watcher in the background and export the PID.
    peerd_pod_watcher 2>&1 &
    local PEERD_POD_WATCHER_PID=$!
    echo $PEERD_POD_WATCHER_PID > peerd_pod_watcher.pid
    echo "Peerd pod watcher started with PID: $PEERD_POD_WATCHER_PID"
}

cmd__peerd_pod_watcher__stop() {
    local PEERD_POD_WATCHER_PID=$(cat peerd_pod_watcher.pid 2>/dev/null)
    if [ -n "$PEERD_POD_WATCHER_PID" ]; then
        echo "Stopping Peerd pod watcher with PID: $PEERD_POD_WATCHER_PID"
        kill $PEERD_POD_WATCHER_PID
        if [ $? -eq 0 ]; then
            echo "Peerd pod watcher stopped successfully."
            rm -f peerd_pod_watcher.pid
        else
            echo "Failed to stop Peerd pod watcher with PID: $PEERD_POD_WATCHER_PID"
        fi
    else
        echo "No Peerd pod watcher process found in file peerd_pod_watcher.pid."
    fi
}

cmd__nodepool__up () {
    local nodepool=$1
    local peerd_image_tag=$PEERD_IMAGE_TAG
    local configureMirrors=$PEERD_CONFIGURE_MIRRORS
    local configureOverlaybdP2p=$PEERD_CONFIGURE_OVERLAYBD_P2P

    ensure_azure_token

    echo "get AKS credentials"
    get_aks_credentials $AKS_NAME $RESOURCE_GROUP

    echo "sanitizing"
    helm uninstall peerd --ignore-not-found=true
    helm uninstall overlaybd-p2p --ignore-not-found=true

    echo "creating new nodepool '$nodepool'"
    nodepool_deploy $AKS_NAME $RESOURCE_GROUP $nodepool $configureOverlaybdP2p

    echo "deploying peerd helm chart using tag '$peerd_image_tag'"
    peerd_helm_deploy $nodepool $peerd_image_tag $configureMirrors

    echo "waiting for pods to connect"
    wait_for_pod_events $AKS_NAME $RESOURCE_GROUP $nodepool "P2PConnected" "app=peerd"
}

cmd__test__llm_streaming() {
    aksName=$AKS_NAME
    rg=$RESOURCE_GROUP
    local nodepool=$1

    echo "running test 'llm_streaming'"

    if [ "$DRY_RUN" == "true" ]; then
        echo "[dry run] would have run test 'llm_streaming'"
    else
        # Deploy the LLM daemonset to the nodepool.
        # Since there is only a single node, this acts as a seeding step, allowing peerd to add this content to its hash table.
        envsubst < $PEERD_LLM_CI_DEPLOY_TEMPLATE | kubectl apply -f -

        # No need to wait for image pull this time, since it's streaming.
         # Scale out the nodepool. This will cause peerd pods to be deployed first, followed by the LLM pods due to the affinity rules.
        az aks nodepool scale --cluster-name $aksName --name $nodepool --resource-group $rg --node-count 4

        wait_for_pod_events $AKS_NAME $RESOURCE_GROUP $nodepool "P2PActive" "app=peerd" 1

        echo "fetching metrics from peerd pods"
        print_peerd_metrics

        echo "cleaning up apps"

        # This will clean up the peerd-ns namespace, which is where the LLM pods are also deployed.
        helm uninstall peerd --ignore-not-found=true
        helm uninstall overlaybd-p2p --ignore-not-found=true

        echo "test 'llm_streaming' complete"
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
