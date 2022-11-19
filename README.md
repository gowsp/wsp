# wsp

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/gowsp/wsp/release)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/gowsp/wsp)
[![Go Report Card](https://goreportcard.com/badge/github.com/gowsp/wsp)](https://goreportcard.com/report/github.com/gowsp/wsp)

wsp 全称**W**eb**S**ocket **P**roxy 是一种基于 WebSocket 的全方位代理, 仅需要web端口即可提供以下功能：

- 正向代理：支持 socks5 http代理，实现突破防火墙的访问
- 反向代理：支持将NAT或防火墙后面的本地服务暴露在Internet
- 网络穿透：将处在NAT设备之后的机器建立连接通讯隧道，进行数据的相互访问

wsp为C/S架构，其中 wsps 位于公网提供 WebSocket 服务，wspc 连接 wsps 进行数据转发，以下为简单的结构示意图

```mermaid
flowchart TD
    wspc1 --> wsps
    wspc2 --> wsps
    wspc3 --> wsps
    wsps --> wspc1
    wsps --> wspc2
    wsps --> wspc3
```

## Wsps

服务端安装，根据操作系统从[Release](https://github.com/gowsp/wsp/releases/latest)下载相应的程序包，解压后将wsps放置在公网机器上，配置用于提供服务的web端口，最小化配置如下：

```json
{
    "port": 8010
}
```

启动服务端， `./wsps -c wsps.json`

### Wspc

wspc功能设计参考了ssh，配置项存在三种转发模式：

- DynamicForward，动态转发，提供正向代理如：socks5，http代理
- RemoteForward，远程转发，将本地端口服务暴露在wsps上，支持 TCP  HTTP HTTPS 协议
- LocalForward，本地转发，用于本地访问已注册的`远程转发`服务

支持连接多个 wsps 服务端，配置格式类似如下：

```json
{
  "client": [
    {
      "server": "ws://wsp1.com:8010",
      "local": []
    }，
    {
      "server": "ws://wsp1.com:8010",
      "remote": []
    }
  ]
}
```

## 正向代理

### client模式

此模式下需要在本机中安装wspc配合wsps使用正向代理，wspc动态转发代理请求，连接的打开和流量传输通过wsps，示意图如下

```mermaid
sequenceDiagram
    actor  client
    participant wspc
    participant wsps
    actor server
    par connect
        wspc --> wsps : keep websocket connect
    and proxy
        client ->> wspc: socks5 or http proxy reqeust
        wspc ->> wsps: request open connect
        wsps ->> server: open connect
        server ->> wsps: connect response
        wsps ->> wspc: connect response
        wspc ->> client: proxy response
    and tranfer data
        client -->> server: tranfer data, via wspc
        server -->> client: tranfer data, via wsps
    end
```

配置格式：`protocols://[bind_address]:port`

- `protocols`支持 socks5 代理协议，HTTP 代理
- `bind_address`可选，空地址表示监听所有网卡IP
- `port`本地监听端口

示例如下：

```json
{
  "client": [
    {
      "server": "ws://mywsps.com:8010",
      "dynamic": [
        "http://:80",
        "socks5://:1080"
      ]
    }
  ]
}
```

启动wspc， `./wsps -c wsps.json`, 此时本地`1080`提供socks5代理，`80`提供http代理, 流量则通过`wsps`进行访问

## 反向代理

将本地服务暴露在wsps上，供浏览器直接访问

### 暴露本地HTTP HTTPS服务

示意图如下：

```mermaid
sequenceDiagram
    actor  browser
    participant wsps
    participant wspc
    actor http as web server
    par register 
        wspc --> wsps: websocket connect
    and connect
        browser ->> wsps : http request
        wsps ->> wspc : select wspc to serve
        wspc ->> http: open connect
        http ->> wspc: connect response
        wspc ->> wsps: connect response
    and reqeust
        wsps ->> wspc: http reqeust data
        wspc ->> http: http reqeust data
        http ->> wspc: http response data
        wspc ->> wsps: http response data
        wsps ->> browser: http response data
    end

```

配置格式：`protocols://bind_address:port/[path]?mode=[mode]&value=[value]`

- `protocols` 支持 http, https（支持websocket）
- `bind_address`http服务地址
- `port`http服务端口
- `path`可选http服务路径
- `mode`访问模式，为以下两种
  - `path` 路径模式
  - `domain` 域名模式

例：

```json
{
  "client": [
    {
      "server": "ws://mywsps.com:8010",
      "remote": [
        "http://127.0.0.1:8080?mode=path&value=api",
        "http://127.0.0.1:8080/api/?mode=path&value=api",
        "http://127.0.0.1:8080?mode=domain&value=customwsp.com",
        "http://127.0.0.1:8080/api/?mode=domain&value=customapi.com"
      ]
    }
  ]
}
```

启动wspc， `./wsps -c wsps.json`，此时在wsps注册的访问映射关系由上至下为

- 访问 http://mywsps.com:8010/api/greet -> http://127.0.0.1:8080/greet
- 访问 http://mywsps.com:8010/api/greet -> http://127.0.0.1:8080/api/greet
- 访问 http://customwsp.com:8010/api/greet -> http://127.0.0.1:8080/api/greet
- 访问 http://customwsp.com:8010/greet -> http://127.0.0.1:8080/api/greet

### 包装本地TCP服务为websocket

某些场景下希望将TCP流量通过websocket进行转发，配置格式：`protocols://bind_address:port?mode=[mode]&value=[value]`

- `protocols` 支持 tcp
- `bind_address`服务IP地址
- `port`服务端口
- `mode`访问模式，为以下两种
  - `path` 路径模式
  - `domain` 域名模式

例暴露vnc服务在websocket，我们配置如下：

```json
{
  "client": [
    {
      "server": "ws://mywsps.com:8010",
      "remote": [
        "tcp://127.0.0.1:5900?mode=path&value=vnc"
      ]
    }
  ]
}
```

这时我们可以打开[novnc](https://novnc.com/noVNC/vnc.html), 修改配置，修改为暴露的vnc服务参数，即可实现vnc的远程访问

## 网络穿透

### 普通模式

此模式下需要两台wspc配合wsps使用，三者功能角色如下：

- wspc client 接入侧
- wsps 中转侧，负责将wspc clinet的代理请求和数据中转给目标的wspc server
- wspc server 服务侧，注册在wsps，负责打开来至wspc client的连接或转发数据

示意图如下

```mermaid
sequenceDiagram
    actor  client
    participant wspc1
    participant wsps
    participant wspc2
    actor target
    par target register 
        wspc2 ->> wsps: websocket connect
    and local register
        wspc1 ->> wsps: websocket connect
    end
    par request connect 
        client ->>+ wspc1: request
        wspc1 ->>+ wsps : request
        wsps ->>+ wspc2 : request
        wspc2 ->>+ target : request
        target ->>- wspc2 : reponse
        wspc2 ->>- wsps : reponse
        wsps ->>- wspc1 : reponse
        wspc1 ->>- client: reponse
    and tranfer data
        client -->> target: virtual connection, via wspc1 websocket
        target -->> client: virtual connection, via wspc2 websocket
    end
```

`wspc server`将本地服务注册至wsps等待`wspc client`连接，配置格式：`protocols://channel[:password]@[bind_address]:port`

- `protocols` 支持 tcp
- `channel`信道标识，注册在wsps上等待其他wspc接入的标识信息
- `password`连接密码，接入的wspc连接密码需要一致才能通讯
- `bind_address`监听地址
- `port`服务端口

如注册本地网络中ssh服务配置如下

```json
{
  "client": [
    {
      "server": "ws://mywsps.com:8010",
      "remote": [
        "tcp://ssh:ssh@192.168.1.200:22"
      ]
    }
  ]
}
```

`wspc client`本地转发，开启本地端口来访问远程已注册的`wspc server`，配置格式：`protocols://remote_channel[:password]@[bind_address]:port`

- `protocols` 支持 tcp
- `channel`信道标识，wsps上已注册的的channel才能访问
- `password`连接密码，与`RemoteForward`端密码一致才能通讯
- `bind_address`监听地址
- `port`本地端口

如访问已注册的`wspc server`ssh服务，本地`wspc client`进行如下配置

```json
{
  "client": [
    {
      "server": "ws://mywsps.com:8010",
      "local": [
        "tcp://ssh:ssh@127.0.0.1:2200"
      ]
    }
  ]
}
```

此时访问本地的`127.0.0.1:2200`即为访问`wspc server`端中`192.168.1.200:22`的ssh服务

### vpn模式

此模式下需要两台wspc配合wsps使用，三者功能角色如下：

- wspc client 代理侧
- wsps 中转侧，负责将wspc clinet的代理请求和数据中转给目标的wspc server
- wspc server 服务侧，注册在wsps，负责打开来至wspc client的代理连接转发代理数据

`wspc server`端配置`tunnel://channel[:password]@`

- `channel`信道标识，注册在wsps上等待其他wspc接入的标识信息
- `password`连接密码，接入的wspc连接密码需要一致才能通讯

例：

```json
{
  "client": [
    {
      "server": "ws://mywsps.com:8010",
      "remote": [
        "tunnel://work_tunnel:password@"
      ]
    }
  ]
}
```

`wspc client`端配置`protocols://remote_channel[:password]@[bind_address]:port`

- `protocols`代理协议，支持 socks5 代理，HTTP 代理
- `remote_channel`信道标识，`RemoteForward`端注册的`channel`
- `password`密码，对应`RemoteForward`端密码
- `bind_address`可选，空地址表示监听所有网卡IP
- `port`本地监听端口

```json
{
  "client": [
    {
      "server": "ws://mywsps.com:8010",
      "dynamic": [
        "socks5://work_tunnel:password@127.0.0.1:1080"
      ]
    }
  ]
}
```

在`wspc client`端用socket5代理的连接和流量都会转发到`wspc server`端，如socket5代理下，访问`192.168.1.200:22`即访问`wspc server`端的`192.168.1.200:22`

## 日志记录

wsp 支持日志分级记录，可在 wsps 或 wspc 中加入如下配置：

```json
{
  "log": {
    "level": "info",
    "output": "/var/log/wsp.log"
  }
}
```

其中 `level` 支持以下级别：

- error：只记录错误
- info：记录信息，包含错误
- debug：记录收发的信令，包含以上信息
- trace：记录全过程信息

## 作为模块引入

wsp在开发时考虑了与现有web服务的协作，支持作为一个功能模块引入

```
go get -u github.com/gowsp/wsp
```

与官方http集成

```go
import "github.com/gowsp/wsp/pkg/server"

config := &server.Config{Auth: "auth"}

server.NewWithHandler(config, http.NewServeMux())
server.NewWithHandler(config, http.DefaultServeMux)
```

与gin集成

```go
import "github.com/gowsp/wsp/pkg/server"

config := &server.Config{Auth: "auth"}
r := gin.Default()
server.NewWithHandler(config, r)
```

## TODO

- [ ] 支持命令行模式使用

## 反馈建议

目前此项目为个人独立开发，难免会有BUG和功能设计上的缺陷，如有问题请提issues反馈，也欢迎参与代码或文档的贡献，祝使用愉快