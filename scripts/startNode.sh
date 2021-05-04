
#!/bin/bash

# MAIN_PATH=main.go

# PORT=6001

# MAX_NODES=50
# COUNTER=0
# while [ $COUNTER -lt $MAX_NODES ]; do
#   go run $MAIN_PATH --port $PORT --first-ip 12.0.0.41:6000 &
#   let COUNTER=COUNTER+1
#   let PORT=PORT+1
#   sleep 0.5
# done

sudo git reset --hard && git checkout feat/test-on-10-regions && sudo git reset --hard && git pull && go run main.go --port 6000 --first-ip 12.0.0.41:6000
