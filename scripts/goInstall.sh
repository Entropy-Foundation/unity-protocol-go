#/bin/bash

curl -O https://storage.googleapis.com/golang/go1.9.3.linux-amd64.tar.gz

tar -xvf go1.9.3.linux-amd64.tar.gz

sudo chown -R root:root ./go
sudo mv go /usr/local
export GOPATH=$HOME/go
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin