#!/bin/bash
binfile=output.rpi
set -e -u -o pipefail -x
env GOOS=linux GOARCH=arm GOARM=6 go build -o $binfile
tar cf - $binfile | ssh dietpi@192.168.24.246 "cd /tmp && tar xf - && ./$binfile -duration 5s"
