### Testcase 1: shutdown server while sending request with curl
open two terminals:
1. terminal 1: $ go run main.go
2. terminal 2: $ curl localhost:8180/
3. terminal 1: $ ctrl + c

Expected: server should be exited after the request is done.

### Testcase 2: shutdown server while uploading a file (>200mb) 
1. $ go run main.go
2. open `./tus-client/index.html` via browser
3. in the browser, choose a file (>200mb) the upload
4. $ ctrl + c

Expected: server should be exited after the upload is successful.

### Testcase 3: shutdown server while uploading a file (>200mb) while try to send a request
1. $ go run main.go
2. open `./tus-client/index.html` via browser
3. in the browser, choose a file (>200mb) the upload
4. $ ctrl + c
5. terminal 2: $ curl localhost:8180/

Expected: 
1. Curl got 505 server not available.
2. Server should be exited after the upload is successful.

# GRACEFUL KAFKA
1. Run kafka container via docker-compose: 
`$ make up`
2. Create topic name is `topic-test` to test
`$ make topic`
3. Access kafka via docker container.
`docker exec -it {container_id} /bin/sh`
4. Send message to `topic-test` in dedicated terminal via docker container.
`kafka-console-producer --bootstrap-server broker:9092 --topic topic-test`
5. Run go server to receive message in other terminal
`go run main.go`
6. Press `ctrl + c` to test

Note: type message `Big` to simulate big job (sleep 5 seconds)