#!/bin/bash
set -e

ensure_azure_token() {
    if [ -z "$SUBSCRIPTION" ]; then
        echo "Error: SUBSCRIPTION is not set."
        exit 1
    fi

    az account set --subscription $SUBSCRIPTION
}

get_az_user() {
    ensure_azure_token
    azUser=$(az account show --query user.name -o tsv | awk -F '@' '{ print $1 }')
    echo -n $azUser
}

