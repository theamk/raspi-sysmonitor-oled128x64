#!/bin/bash
# Builds the monitor, installs it with its systemd unit on the Pi, and (re)starts it.
host=dietpi@192.168.24.246
binfile=output.rpi
unit=raspi-sysmonitor.service
set -e -u -o pipefail -x
env GOOS=linux GOARCH=arm GOARM=6 go build -o $binfile
tar cf - $binfile $unit | ssh $host "
    set -e -u -x
    cd /tmp && tar xf -
    sudo systemctl stop raspi-sysmonitor || true
    sudo install -m 755 $binfile /usr/local/bin/raspi-sysmonitor
    sudo install -m 644 $unit /etc/systemd/system/$unit
    sudo systemctl daemon-reload
    sudo systemctl enable raspi-sysmonitor
    sudo systemctl start raspi-sysmonitor
    sudo systemctl --no-pager status raspi-sysmonitor
"
