#!/bin/bash
binfile=output.rpi
set -e -u -o pipefail -x
env GOOS=linux GOARCH=arm GOARM=6 go build -o $binfile
tar cf - $binfile | ssh dietpi@192.168.24.246 "
    set -e -u
    cd /tmp && tar xf -
    # The service holds the display lock; pause it for the test and restore it afterwards.
    if sudo systemctl is-active --quiet raspi-sysmonitor; then
        sudo systemctl stop raspi-sysmonitor
        trap 'sudo systemctl start raspi-sysmonitor' EXIT
    fi
    ./$binfile -duration 5s
"
