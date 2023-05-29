#!/usr/bin/env bash
set -euo pipefail

## Tool used to import json dumps to vault (e.g., during a migration)

# JSON looks like
# {
#  "pubkey": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
#  "macaroon_hex": "020103aaaaaa",
#  "certificate_base64": "LS0tLS1...",
#  "endpoint": "127.0.0.1:10009",
#

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
