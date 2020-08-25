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
      sudo useradd -u 1001 -g 1001 -M theia
      sudo usermod -a -G vagrant theia 
    }
  SCRIPT

end
