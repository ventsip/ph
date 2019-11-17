.PHONY: build test build_test run

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY=ph
TEST_BINARY=test_process

build:
	cd main; $(GOBUILD) -v -o ../bin/$(BINARY)

test:
	cd lib; $(GOTEST) -v

build_test:
	cd test_process; $(GOBUILD) -v -o ../bin/$(TEST_BINARY)

run: build build_test
	cd bin; (./$(TEST_BINARY) &)
	cd bin; ./$(BINARY)
