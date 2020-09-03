#!/bin/bash

yum groupinstall -y 'Development Tools'
yum install -y gettext-devel openssl-devel perl-CPAN perl-devel zlib-devel curl-devel

# Theia requires a recent version of git
yum -y remove git*
yum -y install wget
export VER="2.27.0"
wget https://github.com/git/git/archive/v${VER}.tar.gz
tar -xvf v${VER}.tar.gz
rm -f v${VER}.tar.gz
cd git-*
make configure
./configure --prefix=/usr
make
make install
