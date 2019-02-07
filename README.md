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
