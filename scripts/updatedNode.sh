#!/bin/bash

git pull
fuser -k 6000/tcp
cd db/
rm -rf *
cd ..