#!/bin/bash
binfile=output.rpi
host=${RPI_HOST:-rpi-laser}
set -e -u -o pipefail -x
env GOOS=linux GOARCH=arm GOARM=6 go build -o $binfile
tar cf - $binfile | ssh $host "
    set -e -u
    cd /tmp && tar xf -
    # The service holds the display lock; pause it for the test and restore it afterwards.
    if sudo systemctl is-active --quiet raspi-sysmonitor; then
        sudo systemctl stop raspi-sysmonitor
        trap 'sudo systemctl start raspi-sysmonitor' EXIT
    fi
    ./$binfile -duration 5s ${*@Q}
"
