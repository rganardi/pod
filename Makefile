VERSION:=$(shell git describe --tags --long --always)
BUILDDATE:=$(shell date "+%FT%T%z")
LDFLAGS=-ldflags "-X main.version_number=${VERSION} -X main.build_date=${BUILDDATE}"

pod: pod.go
	go build ${LDFLAGS} github.com/rganardi/pod

install:
	go install ${LDFLAGS} github.com/rganardi/pod

complete:
	install -m 644 _pod /usr/share/zsh/site-functions
