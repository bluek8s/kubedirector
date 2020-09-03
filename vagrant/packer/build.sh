#!/usr/bin/env bash
# Fail on any error
set -e

# Set version info
BOX_VERSION_BASE=0.0.1

# Set versions requested of main components (These will be used in Packer and passed to Ansible downstream)
export BOX_BASE="centos/7"
export BOX_BASE_VERSION=2004.01
export ANSIBLE_VERSION=2.9.13
export MINIKUBE_VERSION=1.12.3
export DOCKER_VERSION=19.03.12
export KUBECTL_VERSION=1.19.0
export HELM_VERSION=3.3.1
export KUBETAIL_VERSION=1.6.12

# Set versions of supported tools, if they don't match, a warning will be shown on screen
export VIRTUALBOX_VERSION="6.1.12r139181"
export PACKER_VERSION="v1.6.1"
export VAGRANT_VERSION="2.2.9"

# Set the Vagrant cloud user and box name (make sure you have admin permissions to, or are the owner of this repository)
export VAGRANT_CLOUD_BOX_USER="chris-snow"
export VAGRANT_CLOUD_BOX_NAME="kubedirector-lab"

# ############################################################################################## #
# Below this point there should be no need to edit anything, unless you know what you are doing! #
# ############################################################################################## #

echo "Testing if all required tools are installed, please wait..."

# Check if all required tools are installed
if ( ! ( vboxmanage --version >/dev/null 2>&1 && packer version >/dev/null 2>&1 && vagrant version >/dev/null 2>&1 ) )
then
    echo "ERROR: One of the required tools (VirtualBox, Vagrant, and Packer) is not installed. Cannot continue."
    exit 1
fi

# Check the tool versions
INSTALLED_VIRTUALBOX_VERSION=$(vboxmanage --version)
INSTALLED_PACKER_VERSION=$(packer --version)
INSTALLED_VAGRANT_VERSION=$(vagrant --version | awk '{print $2}')

if [[ $INSTALLED_VIRTUALBOX_VERSION != $VIRTUALBOX_VERSION || $INSTALLED_PACKER_VERSION != $PACKER_VERSION || $INSTALLED_VAGRANT_VERSION != $VAGRANT_VERSION ]]
then
    echo "WARNING: One of the tool versions does not match the tested versions. Your mileage may vary..."
    echo " * Using VirtualBox version ${INSTALLED_VIRTUALBOX_VERSION} (tested with version ${VIRTUALBOX_VERSION})"
    echo " * Using Packer version ${INSTALLED_PACKER_VERSION} (tested with version ${PACKER_VERSION})"
    echo " * Using Vagrant version ${INSTALLED_VAGRANT_VERSION} (tested with version ${VAGRANT_VERSION})"
    echo ""
    echo -n "To break, press Ctrl-C now, otherwise press Enter to continue"
    read foo
fi

echo "All required tools found. Continuing."

# Check if a build.env file is present, and if so: source it
if [ -f build.env ]
then
    source build.env
fi

# Check if the variables VAGRANT_CLOUD_USER and VAGRANT_CLOUD_TOKEN have been set, if not ask for them
if [ -z "$DEFAULT_VAGRANT_CLOUD_USER" -o -z "$DEFAULT_VAGRANT_CLOUD_TOKEN" ]
then
    # Ask user for vagrant cloud token
    echo -n "What is your Vagrant Cloud username? [mrvantage] "
    read user
    user=${user:-mrvantage}
    export VAGRANT_CLOUD_USER=${user}

    # Ask user for vagrant cloud token
    echo -n "What is your Vagrant Cloud token? "
    read -s token
    echo ""
    export VAGRANT_CLOUD_TOKEN=${token}
else
    export VAGRANT_CLOUD_USER=$DEFAULT_VAGRANT_CLOUD_USER
    export VAGRANT_CLOUD_TOKEN=$DEFAULT_VAGRANT_CLOUD_TOKEN

    echo "Your vagrant cloud user and token have been sourced from file build.env"
fi

# Export dynamic versioning info
export BOX_VERSION=${BOX_VERSION_BASE}-$(date +'%Y%m%d')
commit=$(git --no-pager log -n 1 --format="%H")
export BOX_VERSION_DESCRIPTION="
## Description
This box is based on the ${BOX_BASE} box version ${BOX_BASE_VERSION}. I try to keep the builds up to date with the latest version of this box.
When the box boots it contains a running minikube, ready to deploy kubenetes manifests, and kubectl is pre configured for the vagrant user.
Helm is installed to allow the immediate deployment of charts.

The box defaults to 2 CPU and 4GB of RAM, it is not advised to limit this.

---

## Versions included in this release
Based on box [${BOX_BASE}](https://app.vagrantup.com/centos/boxes/7) version ${BOX_BASE_VERSION}
* Latest OS updates installed at build time
* ansible ${ANSIBLE_VERSION}
* minikube ${MINIKUBE_VERSION}
* docker ${DOCKER_VERSION}
* kubectl ${KUBECTL_VERSION}
* helm ${HELM_VERSION}
* kubetail ${KUBETAIL_VERSION}

---

$(cat CHANGELOG.md)

---

## Source info
[View source on Github](https://github.com/mrvantage/vagrant-box-centos7-minikube)

Built on commit: \`${commit}\`
"

echo "${BOX_VERSION_DESCRIPTION}"

# Validate build config
echo "Validating build json files"
packer validate packer.json

# Run the actual build
echo "Building box version ${BOX_VERSION}"
packer build -force -on-error=cleanup packer.json

# Tag git commit for this build
git tag -a "${BOX_VERSION}" -m "Version ${BOX_VERSION} built."
