#
# Makefile for dpcimm
#
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
			cat $(CURDIR)/.version 2> /dev/null || echo v0)

PROG = adread
SRCS = $(wildcard *.go)

ifdef NATIVE
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
GOARM =
else
GOOS = linux
GOARCH = arm
GOARM = 5
endif

GOBUILD = GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build
BINDIR = build-$(GOOS)_$(GOARCH)

SUBDIRS =
.PHONY: clean $(SUBDIRS)

all: $(BINDIR)/$(PROG)

$(SUBDIRS):
	$(MAKE) -C $@

$(BINDIR)/$(PROG): $(patsubst %_test.go,,$(SRCS))
	@mkdir -p $(BINDIR)
	$(GOBUILD) -o $(BINDIR)/$(PROG) \
	  -tags release \
	  -ldflags '-X main.Version=$(VERSION) -X main.BuildDate=$(DATE)'

clean:
	rm -rf build-*
	for dir in $(SUBDIRS); do \
	    $(MAKE) -C $$dir clean; \
	done
