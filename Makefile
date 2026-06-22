# BBS Oric — tâches de build, test et déploiement.

.PHONY: help build test vet run studio deploy deploy-build client

help: ## Affiche cette aide
	@echo "BBS Oric — cibles disponibles :"
	@echo "  make build         Compile le serveur (./bbsd)"
	@echo "  make test          Lance tous les tests Go"
	@echo "  make vet           Analyse statique (go vet)"
	@echo "  make run           Lance le serveur en local (0.0.0.0:6502)"
	@echo "  make studio        Lance le studio forge (web, 127.0.0.1:8080)"
	@echo "  make client        Construit la .tap du terminal Oric"
	@echo "  make deploy        Déploie sur le serveur de prod (VPN mustang requis)"
	@echo "  make deploy-build  Compile le binaire de prod sans déployer"

build: ## Compile le serveur
	go build -o bbsd ./server/cmd/bbsd

test: ## Lance les tests
	go test ./...

vet: ## Analyse statique
	go vet ./...

run: build ## Lance le serveur en local
	./bbsd -addr 0.0.0.0:6502

studio: ## Lance le studio forge (éditeur web local)
	go run ./studio/cmd/forge -addr 127.0.0.1:8080

client: ## Construit la .tap du terminal Oric
	./client/build.sh

deploy: ## Déploie le BBS Oric sur le serveur de production
	@./deploy/vps-deploy.sh

deploy-build: ## Compile le binaire de prod (linux/amd64) sans déployer
	@./deploy/vps-deploy.sh --build-only
