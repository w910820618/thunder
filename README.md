## 雷电 

雷电是一个基于UDP协议的发包工具。它是使用Golang语言开发的，得益于golang的协程机制，每秒可以并发多个客户端向单个服务端发送大量请求。

雷电测试的服务器配置为：32g内存 Intel(R) Xeon(R) CPU E5-2670 0 @ 2.60GH。

设置参数 为 18线程、发送64B长度的UDP数据包。

客户端发包测试结果为：
![image](https://github.com/w910820618/picture_repo/blob/master/1567492469993.png)

## 安装及使用说明

### 安装

```
cd  $GOPATH/src
git clone https://github.com/w910820618/thunder.git
cd thunder 
go build *.go
chmod 777 ./client
```

### 运行

- 服务端

```
./client -s -h 127.0.0.1
```

 -s 是否启动服务端，默认为false

-h 服务端绑定的IP地址，默认为127.0.0.1

- 客户端

```
./client -h 127.0.0.1 -n 18 -d 2000s -len 64B
```

-h 服务端绑定的IP地址，默认为127.0.0.1

-n 启动的协程数量

-d 程序运行时间

-len 发送UDP的字节数

