# Vagrant

This document describes how to use [vagrant](https://www.vagrantup.com) to automate setting up a development environment for Kubedirector.

The vagrant environment contains:

 - Minikube 1.18.3
 - Operator SDK 1.5.2
 - Thiea IDE

## Pre-requisites

The following should be installed on your development environment

 - [Virtualbox](https://www.virtualbox.org/)
 - [Vagrant](https://www.vagrantup.com/downloads)

## Usage

### Start environment

Open a terminal, then enter the following:

 ```
 git clone https://github.com/hpe-container-platform-community/kubedirector
 cd kubedirector/vagrant
 vagrant up # this step can take ~ 20 minutes
 vagrant ssh
 ./run_ide.sh
 ```

Open a browser and navigate to: [http://localhost:3000](http://localhost:3000) - this will load the Theia IDE.

Open a terminal in Theia, from here you can:

 - Build and Deploy Kubedirector
 - Use `kubectl` to interact with minikube

### Stop environment

 - Open the terminal where you cloned kubedirector
 - Navigate to the `vagrant` folder
 - Issue `vagrant suspend` 
