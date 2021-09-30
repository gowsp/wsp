# wsp

socks5 and reverse proxy based on websocket

```
            ┌─────────────┐
            │             │
            │    wspc     │
            │             │
            └───┬─────▲───┘
                │     │
                │     │
          ┌─────▼─────┴─────┐
          │    wsps with    │
          │    pubilc ip    │
          └─────┬─────▲─────┘
                │     │
                │     │
            ┌───▼─────┴───┐
            │             │
            │    wspc     │
            │             │
            └─────────────┘
```

## Example Usage

download the latest programs from [Release page](https://github.com/gowsp/wsp/releases)

### server side

Place `wsps` on your public ip server and create `wsps.json` config file

```json
{
    "auth": "auth",
    "path": "/proxy",
    "port": 8010
}
```

start command

```bash
wsps -c wsps.json
```

### client side

Place wspc on the client machine and create `wspc.json` config file

#### socks5 config

```json
{
    "auth": "auth",
    "server": "ws://127.0.0.1:8010/proxy",
    "socks5": ":1080"
}
```

start command

```bash
wspc -c wspc.json
```

### forward local config

if we want forward the local address `192.168.5.16:22` to the `wsps`, we named it `demo`, forward type `local`, the full configuration is as follows

```json
{
    "auth": "auth",
    "server": "ws://127.0.0.1:8010/proxy",
    "addrs": [
        {
            "name": "demo",
            "forward": "local",
            "local_addr": "192.168.5.16",
            "local_port": 22,
            "secret": "secretdemo"
        }
    ]
}
```

### remote visitor config

Another `wspc` accesses the `demo` with type `remote`, Like the following configuration

```json
{
    "auth": "auth",
    "server": "ws://127.0.0.1:8010/proxy",
    "addrs": [
        {
            "name": "demo",
            "forward": "remote",
            "local_addr": "127.0.0.1",
            "local_port": 9909,
            "secret": "secretdemo"
        }
    ]
}
```

When we visit `127.0.0.1:9909` is to visit the `192.168.5.16:22`
