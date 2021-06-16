
#!/bin/bash
PORT=6001

MAX_NODES=150
COUNTER=0
while [ $COUNTER -lt $MAX_NODES ]; do
  # lsof -n -i4TCP:$PORT | grep LISTEN | awk '{ print $2 }' | xargs kill
  sudo kill -9 `sudo lsof -t -i:$PORT`
  let COUNTER=COUNTER+1
  let PORT=PORT+1
done