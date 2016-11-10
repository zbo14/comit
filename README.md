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

## Technical Features 
- asymmetric key cryptography 
- websocket messaging between users 
- bloom filters for subject-specific search
- file submission and compression

## Credits 
logo and artistic consulation from [JFang Design](http://www.jjessfang.com/)


