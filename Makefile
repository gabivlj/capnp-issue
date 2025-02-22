MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MKFILE_DIR  := $(abspath $(dir $(MKFILE_PATH)))
CAPNP_FOLDER := $(MKFILE_DIR)/bytestream
CAPNP ?= capnp
GO ?= go
GO_CAPNP_DIR ?= $(shell go list -m -f '{{.Dir}}' capnproto.org/go/capnp/v3)
BINDIR ?= $(MKFILE_DIR)/bin

$(shell mkdir -p $(BINDIR))

$(BINDIR)/capnpc-go bin/capnpc-go:
	GOBIN=$(dir $(abspath $@)) $(GO) install capnproto.org/go/capnp/v3/capnpc-go@latest
	test -f $(abspath $@) && touch -c $(abspath $@) || exit 1

build: bin/capnpc-go rust
	cd $(CAPNP_FOLDER) && PATH="$$PATH":$(BINDIR) $(CAPNP) compile -I $(GO_CAPNP_DIR)/std/ -I ./ \
	 $(shell find $(CAPNP_FOLDER) -name '*.capnp' | sort -r | while read line; do printf " -ogo "; printf "$$line"; done)
	$(GO) build -o $(BINDIR)/repro .

rust:
	cd rs-client && cargo build --release

repro: build
	bin/repro &
	sleep 1
	rs-client/target/release/cc-test ./repro.sock
