## Thunder

Lightning is a contracting tool based on the UDP protocol. It was developed using the Golang language. Thanks to golang's coroutine mechanism, multiple clients can send multiple requests per second to a single server.

The lightning test server configuration is: 32g memory Intel(R) Xeon(R) CPU E5-2670 0 @ 2.60GH.

Set the parameter to 18 threads and send 64B length UDP packets.

The result of the client packet test is:
![image](https://github.com/w910820618/picture_repo/blob/master/1567492469993.png)

## Installation and use instructions

### Installation

```
cd  $GOPATH/src
git clone https://github.com/w910820618/thunder.git
cd thunder 
go build *.go
chmod 777 ./client
```

### Use instructions

- Server

```
./client -s -h 127.0.0.1
```

 -s Whether to start the server, the default is false

-h IP address bound to the server, the default is 127.0.0.1

- Client

```
./client -h 127.0.0.1 -n 18 -d 2000s -len 64B
```

-h IP address bound to the server. The default is 127.0.0.1.

-n number of coroutines started

-d program runtime

-len The number of bytes sent by UDP

