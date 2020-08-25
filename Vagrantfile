# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|

  config.vm.box = "mrvantage/centos7-minikube"
  config.vm.network "forwarded_port", guest: 3000, host: 3000

  # mount as theia user for theia-ide support
  config.vm.synced_folder ".", "/vagrant", :owner => "1001", :group => "1001" 

  config.vm.provision "shell", inline: <<-SCRIPT
    grep -q theia /etc/group || sudo groupadd -g 1001 theia
    grep -q theia /etc/passwd || { 
      sudo useradd -u 1001 -g 1001 -m theia
      sudo usermod -a -G vagrant theia 
    }
  SCRIPT

  config.vm.provision "shell", inline: <<-SCRIPT
    set -x
    export RELEASE_VERSION=v0.15.2
    export KEY_ID=9391EA2A
    curl -LO -C - https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
    curl -LO -C - https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
    gpg --recv-key "$KEY_ID"
    gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
    chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mkdir -p /usr/local/bin/ && sudo cp operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk && rm operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
    echo
    /usr/local/bin/operator-sdk version
  SCRIPT

end
