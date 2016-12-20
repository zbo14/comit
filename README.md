## Description
Comit provides an infrastructure through which citizens view, submit, and find issues in their area (e.g. service requests, complaints, other types TBD). Submitted text and media (images, video) are appended to a decentralized, censorship-resistant ledger (the blockchain) and sent to a live application feed.

## Installation
- Download and install [Go](https://golang.org/dl/)
- Install [Tendermint](https://github.com/tendermint/tendermint/wiki/Installation)

## Usage
####Demo app
- In terminal window, `cd scripts` from base directory
- Then `sh start_app.sh` to start app
- In another terminal window, `sh start_node.sh` to start tendermint node
- Visit `localhost:8888/endpoint` in your web browser
- See docs for list of endpoints/more details on usage

####Run Tests 
- In terminal window, `cd scripts` from base directory
- Then `sh run_test.sh` to run tests

## Credits 
Design and logo by [JFang Design](http://www.jjessfang.com/)


