.POSIX:
VERSION:=$(shell git describe --tags --long --always)
BUILDDATE:=$(shell date "+%FT%T%z")
LDFLAGS=-ldflags "-X main.version_number=${VERSION} -X main.build_date=${BUILDDATE}"

PREFIX = /usr
MANPREFIX = $(PREFIX)/share/man

.PHONY: install complete docs uninstall

pod: pod.go
	go build ${LDFLAGS} github.com/rganardi/pod

install:
	go install ${LDFLAGS} github.com/rganardi/pod

complete:
	install -m 644 _pod $(DESTDIR)/usr/share/zsh/site-functions

docs:
	sed -e "s/VERSION/${VERSION}/g" < pod.1 > $(DESTDIR)$(MANPREFIX)/man1/pod.1
	chmod 644 $(DESTDIR)$(MANPREFIX)/man1/pod.1

uninstall:
	rm -f $(DESTDIR)/usr/share/zsh/site-functions/_pod
	rm -f $(DESTDIR)$(MANPREFIX)/man1/pod.1
