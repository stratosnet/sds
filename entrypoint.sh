#!/bin/sh

RUN_AS_USER=${RUN_AS_USER:-$(whoami)}
WORK_DIR=${WORK_DIR:-.}
PPD_BIN=${PPD_BIN:-/usr/bin/ppd}
CHAIN_ID=${CHAIN_ID:-tropos-5}
STCHAIN_URL=${STCHAIN_URL:-https://rest-tropos.thestratos.org:443}
SP_LIST=${SP_LIST:-"18.130.202.53:8888 35.74.33.155:8888 52.13.28.64:8888 3.9.152.251:8888 35.73.160.68:8888 18.223.175.117:8888 46.51.251.196:8888"}


if [ ! -d "$WORK_DIR/configs" ]
then
  echo "[entrypoint] Init SDS resource node..."
  printf "\n\n$node\n\n\n\n\nYes" | $PPD_BIN config --create-p2p-key --create-wallet --home $WORK_DIR

  echo "[entrypoint] Set stratos_chain_url as '$STCHAIN_URL'"
  sed -i "s!stratos_chain_url = '.*'!stratos_chain_url = '$STCHAIN_URL'!g" $WORK_DIR/configs/config.toml

  echo "[entrypoint] Set chain_id as '$CHAIN_ID'"
  sed -i "s!chain_id = '.*'!chain_id = '$CHAIN_ID'!g" $WORK_DIR/configs/config.toml

  echo "[entrypoint] Set the default SDS meta node as '$SP_LIST'"
  sed -i "/\[\[sp_list\]\]/,/^network_address = '.*'$/ s/.*//" $WORK_DIR/configs/config.toml
  for sp in $SP_LIST
  do
    cat << EOF >> $WORK_DIR/configs/config.toml
[[sp_list]]
p2p_address = ''
p2p_public_key = ''
network_address = '$sp'
EOF
  done

  echo "[entrypoint] Set the default ports"
  sed -i "s/rest_port = '.*'/rest_port= '9608'/g" $WORK_DIR/configs/config.toml
  sed -i "s/internal_port = '.*'/internal_port = '9708'/g" $WORK_DIR/configs/config.toml
  sed -i "s/rpc_port = '.*'/rpc_port = '8135'/g" $WORK_DIR/configs/config.toml
  sed -i "s/metrics_port = '.*'/metrics_port = '8765'/g" $WORK_DIR/configs/config.toml
  sed -i "s/metrics_port = '.*'/metrics_port = '8765'/g" $WORK_DIR/configs/config.toml
  sed -i "/\[monitor\]/,/^port = '.*'$/ s/^port = '.*'$/port = '5433'/" $WORK_DIR/configs/config.toml
fi

chown -R $RUN_AS_USER $WORK_DIR
su -s /bin/sh - $RUN_AS_USER -c "$@"
