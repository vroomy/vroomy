# Vroomie
Vroomie is a file serving server

## Installation
```bash
go get github.com/Hatch1fy/vroomie
```

## Usage
### With default config
```bash
# With default config (./config.toml)
vroomie
```

### With custom config
```bash
# With custom config
vroomie --config vroomie.toml
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
$ wrk -c60 -d12s https://josh.usehatchapp.com
Running 12s test @ https://josh.usehatchapp.com
  2 threads and 60 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    17.46ms    2.97ms  84.95ms   93.73%
    Req/Sec     1.70k   148.48     2.00k    82.85%
  40655 requests in 12.10s, 4.55GB read
Requests/sec:   3359.25
Transfer/sec:    384.98MB

# vroomie
$ wrk -c60 -d12s https://josh.usehatchapp.com
Running 12s test @ https://josh.usehatchapp.com
  2 threads and 60 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     3.08ms    2.53ms  77.12ms   83.59%
    Req/Sec     9.67k     1.66k   12.09k    88.33%
  231347 requests in 12.07s, 35.08MB read
  Non-2xx or 3xx responses: 231347
Requests/sec:  19167.01
Transfer/sec:      2.91MB
$
```