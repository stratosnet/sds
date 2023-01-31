#!/bin/sh
chown -R $RUN_AS_USER $WORK_DIR
su -s /bin/sh - $RUN_AS_USER -c "$@"
