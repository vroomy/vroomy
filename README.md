# Vroomy
<!-- ALL-CONTRIBUTORS-BADGE:START - Do not remove or modify this section -->
[![All Contributors](https://img.shields.io/badge/all_contributors-3-orange.svg?style=flat-square)](#contributors-)
<!-- ALL-CONTRIBUTORS-BADGE:END -->

![billboard](https://github.com/vroomy/vroomy/blob/main/vroomy-billboard.png?raw=true "Vroomy billboard")
Vroomy is a plugin-based server. Vroomy can be used for anything, from a static file server to a full-blown back-end service!

## Installation
Installing by compilation is very straight forward. The following dependencies are required:
- Go
- GCC

### Fresh Install
If you need to install vroomy use this method! (This installs vroomy, vpm, and all of their dependencies)
```bash
curl -s https://raw.githubusercontent.com/vroomy/vroomy/main/bin/init | bash -s
```

### Self Upgrade
If you already have vroomy installed, it can upgrade itself! (NOTE: this will attempt to self-sign vroomy on osx and support setcap for selinux. For more info, check the directions during install process)
```bash
vroomy upgrade && vpm upgrade
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