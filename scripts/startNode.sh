
#!/bin/bash

MAIN_PATH=main.go

PORT=6001

# # sudo git pull

# MAX_NODES=92
# COUNTER=0
# while [ $COUNTER -lt $MAX_NODES ]; do
#   go run $MAIN_PATH --port $PORT --first-ip 172.17.0.1:6300 &
#   let COUNTER=COUNTER+1
#   let PORT=PORT+1
#   sleep 0.5
# done
# sudo git reset --hard && git checkout feat/test-on-10-regions && sudo git reset --hard && git pull && go run main.go --port 6000 --first-ip 12.0.0.41:6000

# sudo service mongod restart && sudo git reset --hard && git checkout change-structure && sudo git reset --hard && git pull && go get . && go run main.go --port 6000 --first-ip 12.0.0.41:6000
# sudo service mongod restart && sudo git reset --hard && git checkout feat/jayesh/new-DHT-strategy-DHT-node && sudo git reset --hard && git pull && go get . && go run main.go --isDHT true --port 6000 --first-ip 11.1.2.1:6000

go run main.go --port 6000 --first-ip 11.0.3.133:6300