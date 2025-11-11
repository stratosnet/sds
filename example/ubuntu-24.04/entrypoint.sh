#!/bin/sh

RUN_AS_USER=${RUN_AS_USER:-sds}
WORK_DIR=${WORK_DIR:-/sds}
NETWORK_PORT=${NETWORK_PORT:-18081}
PPD_BIN=/usr/bin/ppd


if [ ! -d "$WORK_DIR/config" ]
then
  if [ -z "${MNEMONIC_PHRASE}" ]; then
      echo "Error: The environment variable MNEMONIC_PHRASE is not set." >&2
      exit 1
  fi
  
  if [ -z "${NETWORK_ADDRESS}" ]; then
      echo "Error: The environment variable NETWORK_ADDRESS is not set." >&2
      exit 1
  fi

  echo "[entrypoint] Init SDS resource node..."
  printf "\n\n3\n" | $PPD_BIN config --create-p2p-key --home $WORK_DIR
  printf "\n" | $PPD_BIN config accounts --mnemonic "$MNEMONIC_PHRASE" --home $WORK_DIR

  echo "[entrypoint] Set network_address to '$NETWORK_ADDRESS'"
  sed -i '/\[node\.connectivity\]/,/^\[/ {/network_address/ s/= .*/= '\'$NETWORK_ADDRESS\''/}' $WORK_DIR/config/config.toml

  echo "[entrypoint] Set network_port to $NETWORK_PORT"
  sed -i '/\[node\.connectivity\]/,/^\[/ {/network_port/ s/= .*/= '\'$NETWORK_PORT\''/}' $WORK_DIR/config/config.toml
fi

chown -R $RUN_AS_USER $WORK_DIR
echo "[entrypoint] Starting as user: $RUN_AS_USER"
exec gosu "$RUN_AS_USER" "$@"

