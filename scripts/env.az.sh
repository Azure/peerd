#!/bin/bash
set -e

SUBSCRIPTION="dfb63c8c-7c89-4ef8-af13-75c1d873c895"

ensure_azure_token() {
    az account set --subscription $SUBSCRIPTION
}

get_az_user() {
    ensure_azure_token
    azUser=$(az account show --query user.name -o tsv | awk -F '@' '{ print $1 }')
    echo -n $azUser
}

