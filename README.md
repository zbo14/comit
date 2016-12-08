## Description
Comit provides an infrastructure through which constituents view, submit, and find issues in their local area (e.g. service requests, complaints, other TBD). Submitted information and media (images, video) are appended to an immutable, decentralized ledger (i.e. the blockchain) and broadcast to a live feed.

## Installation
- Download and install [Go](https://golang.org/dl/)
- Install [Tendermint](https://github.com/tendermint/tendermint/wiki/Installation)
- `go get` Tendermint libraries: `go-wire`, `go-crypto`

## Usage
####Demo app
- `cd cmd/comit` from base directory
- `go build` and then `./comit`
- Visit `localhost:8888/path_to_endpoint` in your web browser
- See docs for list of endpoints/more details on usage

####Run Tests 
- `cd app` from base directory
- `go test -v`

## Credits 
Design and logo by [JFang Design](http://www.jjessfang.com/)


