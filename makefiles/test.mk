test:
	go test ./...
	
test-coverage:
	go test ./... -coverprofile=coverage-$(shell basename $(PWD)).out -covermode=count -v
	
.PHONY: test test-coverage