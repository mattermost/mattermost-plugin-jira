.PHONY: test

all: test mattermost-jira-plugin-linux.tar.gz mattermost-jira-plugin-macos.tar.gz mattermost-jira-plugin-windows.tar.gz

mattermost-jira-plugin-linux.tar.gz: $(shell go list -f '{{range .GoFiles}}{{.}} {{end}}') plugin.yaml 
	GOOS=linux GOARCH=amd64 go build -o plugin.exe
	tar -czvf $@ plugin.exe plugin.yaml

mattermost-jira-plugin-macos.tar.gz: $(shell go list -f '{{range .GoFiles}}{{.}} {{end}}') plugin.yaml 
	GOOS=darwin GOARCH=amd64 go build -o plugin.exe
	tar -czvf $@ plugin.exe plugin.yaml

mattermost-jira-plugin-windows.tar.gz: $(shell go list -f '{{range .GoFiles}}{{.}} {{end}}') plugin.yaml 
	GOOS=windows GOARCH=amd64 go build -o plugin.exe
	tar -czvf $@ plugin.exe plugin.yaml

test:
	go test -v ./...
