all: build/shove

PKG    = github.com/majewsky/shove
PREFIX = /usr

GO            = GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR)/build go
GO_BUILDFLAGS =
GO_LDFLAGS    = -s -w

build/shove: FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)/cmd/shove'

# which packages to test with static checkers?
GO_ALLPKGS := $(PKG) $(shell go list $(PKG)/cmd/...)

check: all static-check FORCE
	@printf "\e[1;32m>> All tests successful.\e[0m\n"
static-check: FORCE
	@if ! hash golint 2>/dev/null; then printf "\e[1;36m>> Installing golint...\e[0m\n"; go get -u golang.org/x/lint/golint; fi
	@printf "\e[1;36m>> gofmt\e[0m\n"
	@if s="$$(gofmt -s -l *.go cmd pkg 2>/dev/null)"                            && test -n "$$s"; then printf ' => %s\n%s\n' gofmt  "$$s"; false; fi
	@printf "\e[1;36m>> golint\e[0m\n"
	@if s="$$(golint . && find cmd pkg -type d -exec golint {} \; 2>/dev/null)" && test -n "$$s"; then printf ' => %s\n%s\n' golint "$$s"; false; fi
	@printf "\e[1;36m>> go vet\e[0m\n"
	@$(GO) vet $(GO_ALLPKGS)

install: FORCE all
	install -D -m 0755 build/shove "$(DESTDIR)$(PREFIX)/bin/shove"

.PHONY: FORCE
