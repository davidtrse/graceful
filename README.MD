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