# Vroomy
<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-3-orange.svg?style=flat-square)](#contributors-)
<!-- ALL-CONTRIBUTORS-BADGE:END -->

![billboard](https://github.com/vroomy/vroomy/blob/main/vroomy-billboard.png?raw=true "Vroomy billboard")
Vroomy is a plugin-based server. Vroomy can be used for anything, from a static file server to a full-blown back-end service!

## Installation
To add `vroomy` to your Go project, just call:
`go get github.com/vroomy/vroomy` 

## Getting started
### Example Configuration
```toml
port = 8080
tlsPort = 10443
tlsDir = "./tls"

[env]
fqdn = "https://myserver.org"

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

### Using the library
Getting started with `vroomy` is quite easy! Call `vroomy.New` with the location of your configuration file. For a more in-depth explanation, please check out our [hello-world](https://github.com/vroomy/hello-world) repository.

```go
package main

import (
	"context"
	"log"

	"github.com/vroomy/vroomy"

	_ "github.com/vroomy/hello-world/plugins/companies"
)

func main() {
	var (
		svc *vroomy.Vroomy
		err error
	)

	if svc, err = vroomy.New("./config.toml"); err != nil {
		log.Fatal(err)
	}

	if err = svc.ListenUntilSignal(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

## Usage

### Environment.Get
```go
func ExampleEnvironment_Get() {
	val := exampleEnvironment.Get("foo")
	fmt.Println("Value of foo is", val)
}
```

### Environment.GetInt
```go
func ExampleEnvironment_GetInt() {
	var (
		val int
		err error
	)

	if val, err = exampleEnvironment.GetInt("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.GetInt64
```go
func ExampleEnvironment_GetInt64() {
	var (
		val int64
		err error
	)

	if val, err = exampleEnvironment.GetInt64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.GetFloat64
```go
func ExampleEnvironment_GetFloat64() {
	var (
		val float64
		err error
	)

	if val, err = exampleEnvironment.GetFloat64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.GetTime
```go
func ExampleEnvironment_GetTime() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.GetTime("foo", "2006-01-02"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.GetTimeInLocation
```go
func ExampleEnvironment_GetTimeInLocation() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.GetTimeInLocation("foo", "2006-01-02", time.Local); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.Must
```go
func ExampleEnvironment_Must() {
	var (
		val string
		err error
	)

	if val, err = exampleEnvironment.Must("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.MustInt
```go
func ExampleEnvironment_MustInt() {
	var (
		val int
		err error
	)

	if val, err = exampleEnvironment.MustInt("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.MustInt64
```go
func ExampleEnvironment_MustInt64() {
	var (
		val int64
		err error
	)

	if val, err = exampleEnvironment.MustInt64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.MustFloat64
```go
func ExampleEnvironment_MustFloat64() {
	var (
		val float64
		err error
	)

	if val, err = exampleEnvironment.MustFloat64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.MustTime
```go
func ExampleEnvironment_MustTime() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.MustTime("foo", "2006-01-02"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

### Environment.MustTimeInLocation
```go
func ExampleEnvironment_MustTimeInLocation() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.MustTimeInLocation("foo", "2006-01-02", time.Local); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
```

## Flags

### [-dataDir -d]
  :: Initializes backends in provided directory.
  Overrides value set in config and default values.
  Ignored when testing in favor of dir "testData".  
  Use `vroomy -d <dir>`

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tr>
    <td align="center"><a href="http://itsmontoya.com"><img src="https://avatars2.githubusercontent.com/u/928954?v=4" width="100px;" alt=""/><br /><sub><b>Josh</b></sub></a><br /><a href="https://github.com/vroomy/vroomy/commits?author=itsmontoya" title="Code">💻</a> <a href="https://github.com/vroomy/vroomy/commits?author=itsmontoya" title="Documentation">📖</a></td>
    <td align="center"><a href="https://github.com/dhalman"><img src="https://avatars3.githubusercontent.com/u/1349742?v=4" width="100px;" alt=""/><br /><sub><b>Derek Halman</b></sub></a><br /><a href="https://github.com/vroomy/vroomy/commits?author=dhalman" title="Code">💻</a></td>
    <td align="center"><a href="http://mattstay.com"><img src="https://avatars0.githubusercontent.com/u/414740?v=4" width="100px;" alt=""/><br /><sub><b>Matt Stay</b></sub></a><br /><a href="#design-matthew-stay" title="Design">🎨</a></td>
  </tr>
</table>

<!-- markdownlint-enable -->
<!-- prettier-ignore-end -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

This project follows the [all-contributors](https://github.com/all-contributors/all-contributors) specification. Contributions of any kind welcome!