BINARY=gomake

VERSION=1.0.0
BUILD=`git rev-parse HEAD`

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

.DEFAULT_GOAL: ${BINARY}

build:
	go build ${LDFLAGS} -o ${BINARY} github.com/zballs/comit/cmd/comit

install:
	go install ${LDFLAGS} github.com/zballs/comit/cmd/comit

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

.PHONY: clean install 

