# wsp

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/gowsp/wsp/release)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/gowsp/wsp)
[![Go Report Card](https://goreportcard.com/badge/github.com/gowsp/wsp)](https://goreportcard.com/report/github.com/gowsp/wsp)

wsp 全称**W**eb**S**ocket **P**roxy 是一种基于 WebSocket 的全方位代理, 仅需要web端口即可提供以下功能：

- 正向代理：支持 socks5，实现突破防火墙的访问
- 反向代理：支持将NAT或防火墙后面的本地服务器暴露给Internet

wsp为C/S架构，其中 wsps 位于公网提供 WebSocket 服务， wspc 连接 wsps 进行数据转发，以下为简单的结构示意图

```
  ┌─────────────┐
  │             │
  │    wspc     │
  │             │
  └───┬─────▲───┘
      │     │
      │     │
┌─────▼─────┴─────┐
│                 │
│     wsps web    │
│                 │
└─────┬─────▲─────┘
      │     │
      │     │
  ┌───▼─────┴───┐
  │             │
  │    wspc     │
  │             │
  └─────────────┘
```

## Wsps

服务端安装，下载[Release page](https://github.com/gowsp/wsp/releases/latest)的程序，将wsps放置在公网机器，配置用于提供web服务的端口，其最小化配置如下：

```json
{
    "port": 8010
}
```

启动服务端， `./wsps -c wsps.json`

### Wspc

wspc功能设计参考了ssh，提供以下三种转发模式：

- DynamicForward，动态转发， 提供正向代理如：socks5，http代理
- RemoteForward，远程转发，将本地端口连接转发至远端，支持 TCP UDP HTTP HTTPS 协议
  - HTTP 和 HTTPS 直接注册在wsps的web服务上，支持域名和路径两种方式
  - TCP UDP 注册在 wsps 后等待其他 wspc 端的接入
- LocalForward，本地转发，用于本地访问已注册的`远程转发`服务

## DynamicForward

正向代理，动态转发连接请求，配置格式：`protocols://[bind_address]:port`

- `protocols`支持 socks5 代理协议，HTTP 代理
- `bind_address`可选，空地址表示监听所有网卡IP
- `port`本地监听端口

示例如下：

```json
{
    "server": "ws://mywsps.com:8010",
    "dynamic": [
        "http://:80"，
        "socks5://:1080"
    ]
}
```

启动wspc， `./wsps -c wsps.json`, 此时本地`1080`即提供socks5代理, 流量则通过`wsps`进行访问

## RemoteForward

远程转发，将本地服务暴露在wsps上，供远程wspc或浏览器访问

### TCP UDP

配置格式：`protocols://channel[:password]@[bind_address]:port`

- `protocols` 支持 tcp, udp
- `channel`信道标识，注册在wsps上等待其他wspc接入的标识信息
- `password`连接密码，接入的wspc连接密码需要一致才能通讯
- `bind_address`监听地址
- `port`服务端口

该转发主要与`LocalForward`配合使用，如暴露本地网络中ssh服务配置如下

```json
{
    "server": "ws://mywsps.com:8010",
    "remote": [
        "tcp://ssh:ssh@192.168.1.200:22"
    ]
}
```

### HTTP HTTPS

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
    "server": "ws://mywsps.com:8010",
    "remote": [
        "http://127.0.0.1:8080?mode=path&value=api",
        "http://127.0.0.1:8080/api?mode=path&value=api",
        "http://127.0.0.1:8080?mode=domain&value=customwsp.com",
        "http://127.0.0.1:8080/api?mode=domain&value=customapi.com",
    ]
}
```

启动wspc， `./wsps -c wsps.json`，此时在wsps注册的访问映射关系由上至下为

- 访问 http://mywsps.com:8010/api/greet -> http://127.0.0.1:8080/greet
- 访问 http://mywsps.com:8010/api/greet -> http://127.0.0.1:8080/api/greet
- 访问 http://customwsp.com:8010/api/greet -> http://127.0.0.1:8080/api/greet
- 访问 http://customwsp.com:8010/greet -> http://127.0.0.1:8080/api/greet

## RemoteForward && LocalForward

`LocalForward`本地转发，开启本地端口来访问远程已注册的`RemoteForward`，配置格式：`protocols://remote_channel[:password]@[bind_address]:port`

- `protocols` 支持 tcp, udp
- `channel`信道标识，wsps上已注册的的channel才能访问
- `password`连接密码，与`RemoteForward`端密码一致才能通讯
- `bind_address`监听地址
- `port`本地端口

如`RemoteForward`端wspc开启了ssh远程转发配置如下

```json
{
    "server": "ws://mywsps.com:8010",
    "remote": [
        "tcp://ssh:ssh@192.168.1.200:22"
    ]
}
```

本地`LocalForward`进行连接则需要进行如下配置，

```json
{
    "server": "ws://mywsps.com:8010",
    "local": [
        "tcp://ssh:ssh@127.0.0.1:2200"
    ]
}
```

此时访问本地的`127.0.0.1:2200`即为访问`RemoteForward`端中`192.168.1.200:22`的ssh服务

## RemoteForward && DynamicForward

`RemoteForward && LocalForward`虽已满足了大部分场景，但在配置上`RemoteForward`端开放一个端口，`LocalForward`端也要开放端口，在使用上带来了一些不便，将`RemoteForward`与`DynamicForward`结合可以实现地址的动态打开，带来类似vpn的体验

`RemoteForward`端配置`tunnel://channel[:password]@`

- `channel`信道标识，注册在wsps上等待其他wspc接入的标识信息
- `password`连接密码，接入的wspc连接密码需要一致才能通讯

例：

```json
{
    "server": "ws://mywsps.com:8010",
    "remote": [
        "tunnel://work_tunnel:password@"
    ]
}
```

`DynamicForward`端配置`protocols://remote_channel[:password]@[bind_address]:port`

- `protocols`代理协议，支持 socks5 代理，HTTP 代理
- `remote_channel`信道标识，`RemoteForward`端注册的`channel`
- `password`密码，对应`RemoteForward`端密码
- `bind_address`可选，空地址表示监听所有网卡IP
- `port`本地监听端口

```json
{
    "server": "ws://mywsps.com:8010",
    "local": [
        "socks5://work_tunnel:password@127.0.0.1:1080"
    ]
}
```

在`DynamicForward`端用socket5代理的连接和流量都会转发到`RemoteForward`端，如socket5代理下，访问`192.168.1.200:22`即访问`RemoteForward`端的`192.168.1.200:22`

## 作为模块引入

wsp在开发时考虑了与现有web服务的协作，支持作为一个功能模块引入

```
go get -u https://github.com/gowsp/wsp
```

与官方http集成

```go
import "github.com/gowsp/wsp/pkg/server"

config := &server.Config{Auth: "auth"}

server.NewWspsWithHandler(config, http.NewServeMux())
server.NewWspsWithHandler(config, http.DefaultServeMux)
```

与gin集成

```go
import "github.com/gowsp/wsp/pkg/server"

config := &server.Config{Auth: "auth"}
r := gin.Default()
server.NewWspsWithHandler(config, r)
```

## TODO

- [x] DynamicForward支持http代理协议
- [x] DynamicForward支持远端动态连接
- [ ] TCP UDP支持ws直接暴露
- [ ] 消息处理事件驱动设计
- [ ] 传输流量加密处理
- [ ] 支持命令行模式使用

## 反馈建议

目前此项目为个人独立开发，难免会有BUG和功能设计上的缺陷，如有问题请提issues反馈，祝你使用愉快