GC := go build
VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
Minversion := $(shell date)
BUILD_NODE_PAR := -ldflags "-X github.com/elastos/Elastos.ELA.Arbiter/config.Version=$(VERSION)" #-race

all:
	$(GC) $(BUILD_NODE_PAR) -o arbiter arbiter.go

clean:
	rm -rf *.8 *.o *.out *.6 .*.swp