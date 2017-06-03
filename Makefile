VERSION:=$(shell git describe --tags --long --always)
BUILDDATE:=$(shell date "+%FT%T%z")
LDFLAGS=-ldflags "-X main.version_number=${VERSION} -X main.build_date=${BUILDDATE}"

pod: pod.go
	go build ${LDFLAGS}

install:
	go install ${LDFLAGS}
