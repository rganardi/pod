VERSION:=$(shell git describe --tags --long --always)
BUILDDATE:=$(shell date "+%FT%T%z")
LDFLAGS=-ldflags "-X main.version_number=${VERSION} -X main.build_date=${BUILDDATE}"

.PHONY: install complete docs uninstall

pod: pod.go
	go build ${LDFLAGS} github.com/rganardi/pod

install:
	go install ${LDFLAGS} github.com/rganardi/pod

complete:
	install -m 644 _pod /usr/share/zsh/site-functions

docs:
	sed -e "s/VERSION/${VERSION}/g" < pod.1 > /usr/share/man/man1/pod.1
	chmod 644 /usr/share/man/man1/pod.1

uninstall:
	rm -f /usr/share/zsh/site-functions/_pod
	rm -f /usr/share/man/man1/pod.1
