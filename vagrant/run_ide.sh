#!/bin/bash

if [[ $(whoami) != "vagrant" || $(hostname) != "control-plane.minikube.internal" ]];
then
    echo "Aborting. This script should be run in vagrant box by vagrant user."
    exit 1
fi

# configure Theia
SHELL=/bin/bash \
THEIA_DEFAULT_PLUGINS=local-dir:/vagrant/vagrant/plugins  \
GOPATH=/home/project
PATH=$PATH:$GOPATH/bin

cd /home/vagrant
yarn
yarn theia build

node ./src-gen/backend/main.js /vagrant --hostname=0.0.0.0
