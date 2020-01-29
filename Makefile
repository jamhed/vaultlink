.PHONY: skaffold

skaffold:
	cd src && GOOS=linux go build -o ../tele/vaultlink
