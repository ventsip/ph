.PHONY: build test build_test run

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY=ph
TEST_BINARY=test_process

build:
	cd test; $(GOBUILD) -v -o $(BINARY)

test:
	$(GOTEST) -v

build_test:
	cd test_process; $(GOBUILD) -v -o $(TEST_BINARY)

run: build build_test
	test_process/$(TEST_BINARY) &
	test/$(BINARY)
