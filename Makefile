-include .env

env:
	echo "GITHUB_READ_TOKEN=$(shell kubectl get secret --context=dev-gcp --namespace=nada github-read-token -o jsonpath='{.data.GITHUB_READ_TOKEN}' | base64 -d)" >> .env

local:
	GITHUB_READ_TOKEN=$(GITHUB_READ_TOKEN) go run .
