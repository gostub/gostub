# gostub

Simple http stub server by golang

![gostub](./gostub.png)

## Requirement

Only Go language

## Installation

```sh
go get github.com/gostub/gostub

```

## Usage

```
$ gostub -h

Usage of gostub:
  -o string
    	output path (e.g. 'tests' -> ./tests)
  -p string
    	port number (default "8181")
```

## Example

### Add route `GET /hello/world`

```
.
└── hello
    └── world
        ├── $GET.json
        └── response.json
```

**$GET.json**

```json
{
  "default" : {
    "body": "response.json",
    "status": 200
  }
}
```

**response.json**

```
{
  "hello": "world!"
}
```

### Response

**launch**

```
$ gostub
```

**curl**

```sh
$ curl http://localhost:8181/hello/world

{
  "hello": "world!"
}
```

### Shutdown

```
$ curl http://localhost:8181/gostub/shutdown
```
