#!/bin/bash

if [[ $(whoami) != "vagrant" || $(hostname) != "control-plane.minikube.internal" ]];
then
    echo "Aborting. This script should be run in vagrant box by vagrant user."
    exit 1
fi

docker run --privileged=true -it -p 3000:3000 -v "/vagrant:/home/project:Z,cached" theiaide/theia-go
