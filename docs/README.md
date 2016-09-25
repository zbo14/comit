## Installation
- Download and install [Go](https://golang.org/dl/)
- Install [Tendermint](https://github.com/tendermint/tendermint/wiki/Installation)
- In terminal window
	- `go get github.com/tendermint/go-crypto` 
	- `go get github.com/tendermint/go-p2p`
	- `go get github.com/zballs/3ii`

## Usage
####Demo app
- In terminal window
  - From base directory, `cd main/3ii/`
  - `go build` 
  - `./3ii`
  - Visit `localhost:8888/path_to_endpoint` in your web browser
  - See docs for list of endpoints and more details on usage


####Run Tests
- In terminal window 
  - From base directory, `cd actions/`
  - `go test -v`
