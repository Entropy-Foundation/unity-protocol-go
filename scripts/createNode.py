f= open("change.txt","a+")
port=6000

for i in range(101):
    f.write("%d.node:\n\t build: \n\t\t context: . \n\t\t dockerfile: ./Dockerfile \n\t depends_on: \n\t - \"boot.node\" " %(i))
    if i > 1 :
        for j in range(i-1):
            j+=1
            f.write("\n\t - \"%d.node\" "%(j)) 
        

    f.write("\n\t environment: \n\t\t port: %d \n\t\t first-ip: 127.0.1.1:6000 \n\t\t IS_COMPOSE: \"true\" \n\t mem_limit: 300m \n\t ports: \n\t\t - \"%d:%d\" \n"%(port,port,port))
    port+=1
f.close()
