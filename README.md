# wsp

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/gowsp/wsp/release)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/gowsp/wsp)

Websocket-based socks5 and reverse proxy

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
## Example Usage

download the latest programs from [Release page](https://github.com/gowsp/wsp/releases) page according to your operating system and architecture.

Put `wsps` and `wsps.json` onto your server A with public IP.

Put `wspc` and `wspc.json` onto your server B in LAN (that can't be connected from public Internet).

### socks5 proxy

1. Modify `wsps.json` on server A and set the `port` to be connected to wsp clients:

  ```json
  {
      "port": 8000
  }
  ```

2. Start `wsps` on server A:

  `./wsps -c ./wsps.json`

3. On server B, modify `wspc.json` to put in your `wsps` server websocket url as `server` field:

  ```json
  {
      "server": "ws://x.x.x.x:8000",
      "socks5": ":1080"
  }
  ```
4. Start `wspc` on server B:

  `./wspc -c ./wspc.json`

5. Now We will forward the socket5 proxy to server A via websocket

### Visit your web service in LAN

We can expose an local web service behind a NAT network to others for testing by using wsp.

1. Modify `wsps.json` on server A and set the `port` to be connected to wsp clients:

  ```json
  {
      "port": 8000
  }
  ```

2. Start `wsps`:

  `./wsps -c ./wsps.json`

3. Modify `wspc.json` and set `server` to the websocket address of the remote wsps server. The `local_port` is the port of your web service, `name` as a unique identifier will be the prefix for our access to web services, For this example we use `api`: 

  ```json
  {
      "auth": "auth",
      "server": "ws://x.x.x.x:8000",
      "addrs": [
          {
              "forward": "http",
              "name": "api",
              "local_addr": "127.0.0.1",
              "local_port": 8080
          }
      ]
  }
  ```

4. Start `wspc`:

  `./wspc -c ./wspc.json`

5. Now visit your local web service using url `http://x.x.x.x:8000/api`.

### Access your computer in LAN by SSH

We will place the ssh connection on wsps for access on the other end
### server

`wsps` config like before

  ```json
  {
      "port": 8000
  }
  ```

#### local config

The ssh client side we named it `demo`, forward type `local`, the full configuration is as follows

```json
{
    "server": "ws://127.0.0.1:8000",
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

Start `wspc`:

  `./wspc -c ./wspc.json`

#### remote config

Another `wspc` accesses the `demo` with type `remote`, Like the following configuration

```json
{
    "server": "ws://127.0.0.1:8000",
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

Start `wspc`:

  `./wspc -c ./wspc.json`

When we visit `127.0.0.1:9909` is to visit the `192.168.5.16:22`

### Enable Authentication

Enable authentication by setting auth on the wsps side

```json
{
    "auth": "auth",
    "port": 8000
}
```

The same requires us to set up authentication on the wspc side as well

```json
{
    "auth": "auth",
    "server": "ws://x.x.x.x:8000",
    "socks5": ":1080"
}
```

### Custom path

We can customize the path of ws

```json
{
    "auth": "auth",
    "path": "/proxy",
    "port": 8000
}
```

The same clients also need to be consistent

```json
{
    "auth": "auth",
    "server": "ws://x.x.x.x:8000/proxy",
    "socks5": ":1080"
}
```