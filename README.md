# Vroomy

![billboard](https://github.com/vroomy/vroomy/blob/master/vroomy-billboard.png?raw=true "Vroomy billboard")
Vroomy is a plugin-based server. Vroomy can be used for anything, from a static file server to a full-blown back-end service!

## Installation
Installing by compilation is very straight forward. The following dependencies are required:
- Go
- GCC

### Method 1 - Fresh Install
If you need to install vroomy:
```bash
go get github.com/vroomy/vpm
go get github.com/vroomy/vroomy

cd ~/go/src/github.com/vroomy/vroomy
go install -trimpath

cd ~/go/src/github.com/vroomy/vpm
go install -trimpath
```

### Method 2 - Self Upgrade
If you already have vroomy installed:
```bash
vroomy upgrade && vpm upgrade

```

## Usage
### Start (with default config)
```bash
# With default config (./config.toml)
vroomy
```

### Start (with custom config)
```bash
# With custom config
vroomy --config custom.toml
```

### Update plugins
```bash
vpm update
```

## Example configuration
```toml
port = 8080
tlsPort = 10443 
tlsDir = "./tls"

[[route]]
httpPath = "/"
target = "./public_html/index.html"

[[route]]
httpPath = "/js/*"
target = "./public_html/js"

[[route]]
httpPath = "/css/*"
target = "./public_html/css"
```

*Note: Please see config.example.toml for a more in depth example*

### Performance
```bash
# nginx
$ wrk -c60 -d20s https://josh.usehatchapp.com
Running 20s test @ https://josh.usehatchapp.com
  2 threads and 60 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    17.66ms    1.59ms  30.31ms   88.20%
    Req/Sec     1.68k   102.72     1.91k    85.43%
  66500 requests in 20.01s, 7.44GB read
Requests/sec:   3323.69
Transfer/sec:    380.91MB

# vroomy
$ wrk -c60 -d20s https://josh.usehatchapp.com
Running 20s test @ https://josh.usehatchapp.com
  2 threads and 60 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    14.28ms    9.45ms  98.77ms   73.79%
    Req/Sec     2.17k   304.46     3.03k    76.52%
  86013 requests in 20.01s, 9.62GB read
Requests/sec:   4297.88
Transfer/sec:    492.22MB
```

## Dynamic Commands/Flags
Commands and flags can be added in config.toml and will automatically print in `vroomy help`
`handler` represents the plugin.method of a command handler.

```
Example: 
[[command]]
name = "seed"
usage = "Use `vroomy seed` to execute the seed plugin handler\n  Accepts flag -seedfile <filepath>"
handler = "seed.Reseed"

[[flag]]
name = "seedfile"
defaultValue = "test"
usage = "Set the seed file (i.e. \"custom.json\" when you want to run a custom seed"
```

## Default Commands

These are provided by default and are "reserved" commands. They cannot be used in dynamic configs.
:: vroomy :: Usage ::

### vroomy
  :: Runs vroomy server.
  Accepts flags specified in config.toml.
  Use `vroomy` or `vroomy -<flag>`

### vroomy test
  :: Tests the currently built plugins for compatibility.
  Closes service upon successful execution. Avoids port binding.
  Use `vroomy test`

### vroomy help
  :: Prints available commands and flags.
  Use `vroomy help <command>` or `vroomy help <-flag>` to get more specific info.

### vroomy version
  :: Prints current version of vroomy installation.
  Use `vroomy version`

### vroomy upgrade
  :: Upgrades vroomy installation itself.
  Skips if version is up to date.
  Use `vroomy upgrade` or `vroomy upgrade <branch>`

## Flags

### [-require -r]
  :: Initializes only the specified "required" plugins.
  Allows optimized custom commands.
  Use `vroomy test -r <plugin> <plugin>`
