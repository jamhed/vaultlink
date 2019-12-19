.PHONY: skaffold

skaffold:
	cd src && GOOS=linux go build -o ../skaffold/vaultlink
