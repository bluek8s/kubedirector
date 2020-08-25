#!/bin/bash

if [[ $(whoami) != "vagrant" || $(hostname) != "control-plane.minikube.internal" ]];
then
    echo "Aborting. This script should be run in vagrant box by vagrant user."
    exit 1
fi

docker stop "$(docker ps | grep theiaide/theia-go | head -n1 | awk '{ print $1 }')"
