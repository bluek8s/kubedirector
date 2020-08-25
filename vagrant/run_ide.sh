#!/bin/bash

if [[ $(whoami) != "vagrant" || $(hostname) != "control-plane.minikube.internal" ]];
then
    echo "Aborting. This script should be run in vagrant box by vagrant user."
    exit 1
fi

SHELL=/bin/bash
THEIA_DEFAULT_PLUGINS=local-dir:/home/vagrant/plugins 
GOPATH=/home/project
PATH=$PATH:$GOPATH/bin

cd /home/vagrant

node ./src-gen/backend/main.js /vagrant --hostname=0.0.0.0
