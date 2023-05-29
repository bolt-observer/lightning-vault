#!/usr/bin/env bash

set -euo pipefail

for file in *.json; do
  one=$(basename $file | cut -d "." -f 1)
  name=$(cut -d "_" -f 1 <<< $one)
  unique_id=$(cut -d "_" -f 2 <<< $one)
  if [ "$name" == "$unique_id" ]; then
    unique_id=""
  fi
  if [ "$unique_id" == "" ]; then
    echo curl -Lvvv -X POST -u write -d@$file $MACAROON_STORAGE_URL/put/?verify=false
    curl -Lvvv -X POST -u write -d@$file $MACAROON_STORAGE_URL/put/?verify=false
  else
    echo curl -Lvvv -X POST -u write -d@$file $MACAROON_STORAGE_URL/put/$unique_id?verify=false
    curl -Lvvv -X POST -u write -d@$file $MACAROON_STORAGE_URL/put/$unique_id?verify=false
  fi
done
