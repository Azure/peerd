#!/bin/bash
set -e

SUBSCRIPTION=""

ensure_azure_token() {
    az account set --subscription $SUBSCRIPTION
}

get_az_user() {
    ensure_azure_token
    azUser=$(az account show --query user.name -o tsv | awk -F '@' '{ print $1 }')
    echo -n $azUser
}

