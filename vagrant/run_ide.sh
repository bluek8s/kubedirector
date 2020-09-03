#!/bin/bash

if [[ $(whoami) == "vagrant" || $(hostname) == "control-plane.minikube.internal" ]];
then
    echo "Aborting. This script should not be run in vagrant box by vagrant user."
    exit 1
fi

git_vars=1
if [[ -z $GIT_USER ]]; then
  echo "GIT_USER variable not found"
  git_vars=0
else
  
  CURRENT_GIT_USER=$(git config credential.https://github.com.username)
  if [[ "$CURRENT_GIT_USER" != "$GIT_USER" ]]; then

    echo "Found username '${CURRENT_GIT_USER}' in project git config."
    echo "Found username '${GIT_USER}' in GIT_USER environment variable."

    while true; do
        read -p "Would you like to update your git config to '${GIT_USER}'?" yn
        case $yn in
            [Yy]* ) git config credential.https://github.com.username $GIT_USER; break;;
            [Nn]* ) break;;
            * ) echo "Please answer yes or no.";;
        esac
    done
  fi
fi

if [[ -z $GIT_PASS ]]; then
  echo "GIT_PASS variable not found"
  git_vars=0
fi

if [[ -z $GIT_AUTHOR_NAME ]]; then
  echo "GIT_AUTHOR_NAME variable not found"
  git_vars=0
fi

if [[ -z $GIT_COMMITTER_NAME ]]; then
  echo "GIT_COMMITTER_NAME variable not found"
  git_vars=0
fi

if [[ -z $GIT_AUTHOR_EMAIL ]]; then
  echo "GIT_AUTHOR_EMAIL variable not found"
  git_vars=0
fi

if [[ -z $GIT_COMMITTER_EMAIL ]]; then
  echo "GIT_COMMITER_EMAIL variable not found"
  git_vars=0
fi

if [[ $git_vars == 0 ]]; then
  echo
  echo "WARNING:"
  echo "One or more git variables were not set."
  echo "You will not be able to push/pull to github from inside theia."
  echo
  echo "TIP:"
  echo "you can set these variables in .bashrc or .bash_profile, e.g."
  echo -------------------------------------
  echo export GIT_USER=your_git_username
  echo export GIT_PASS=your_git_password
  echo export GIT_AUTHOR_NAME="Your name"
  echo export GIT_COMMITTER_NAME="Your name"
  echo export GIT_AUTHOR_EMAIL=your@email
  echo export GIT_COMMITTER_EMAIL=your@email
  echo -------------------------------------
  echo

  while true; do
      read -p "Do you want to continue without git configured in Theia?" yn
      case $yn in
          [Yy]* ) break;;
          [Nn]* ) exit;;
          * ) echo "Please answer yes or no.";;
      esac
  done
fi

vagrant ssh -c "
    export SHELL=/bin/bash
    export THEIA_DEFAULT_PLUGINS=local-dir:/home/vagrant/plugins 
    export GOPATH=/home/project
    export PATH=$PATH:$GOPATH/bin
    
    # set env variables from local environment
    export GIT_USER='$GIT_USER'
    export GIT_PASS='$GIT_PASS'
    export GIT_AUTHOR_NAME='$GIT_AUTHOR_NAME'
    export GIT_COMMITTER_NAME='$GIT_COMMITTER_NAME'
    export GIT_AUTHOR_EMAIL='$GIT_AUTHOR_EMAIL'
    export GIT_COMMITTER_EMAIL='$GIT_COMMITTER_EMAIL'
    export GIT_ASKPASS=/home/vagrant/git_env_password.sh
    
    cd /home/vagrant
    
    node ./src-gen/backend/main.js /vagrant/src/github.com/bluek8s/kubedirector/ --hostname=0.0.0.0
"
