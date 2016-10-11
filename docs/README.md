## Installation
- Download and install [Go](https://golang.org/dl/)
- Install [Tendermint](https://github.com/tendermint/tendermint/wiki/Installation)
- `go get` Tendermint libraries: `go-wire`, `go-crypto`, `go-p2p`

## Usage
####Demo app
- `cd main/3ii` from base directory
- `go build` and then `./3ii`
- Visit `localhost:8888/path_to_endpoint` in your web browser
- See docs for list of endpoints/more details on usage

####Run Tests 
- `cd app` from base directory
- `go test -v`
