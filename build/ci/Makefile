.PHONY: tests-deps-install
tests-deps-install: ## Install dependencies for testing (supported only on Ubuntu)
	@echo "+ $@"
	sudo $(SCRIPTS_DIR)/install-deps.sh

.PHONY: kind-create
kind-create: ## Creates a kind cluster
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster create -y )

.PHONY: kind-get
kind-get: ## Shows the current kind cluster
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster get -y )

.PHONY: kind-delete
kind-delete: ## Deletes kind cluster
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster delete -y )

.PHONY: kind-deploy
kind-deploy: ## Deploys the p2p application to kind cluster
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	@( $(SCRIPTS_DIR)/kind.sh app deploy -y $(CONTAINER_REGISTRY)/peerd:dev )

.PHONY: kind-test-ctr
kind-test-ctr: ## Deploys test 'ctr' to the kind cluster
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh test ctr -y )

.PHONY: kind-test-random
kind-test-random: ## Deploys test 'random' to the kind cluster
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	@( $(SCRIPTS_DIR)/kind.sh test random -y $(CONTAINER_REGISTRY)/random:dev )

.PHONY: tests-ci-aks
tests-ci-aks: ## Run CI in AKS
	@echo "+ $@"
ifndef NODEPOOL
	$(eval NODEPOOL := $(shell date +"p2p%y%m%d"))
endif
ifndef PEERD_IMAGE_TAG
	$(eval PEERD_IMAGE_TAG := "dev")
endif
	@echo "\033[92mRunning CI NODEPOOL: $(NODEPOOL)\033[0m"
	@( PEERD_IMAGE_TAG=$(PEERD_IMAGE_TAG) $(SCRIPTS_DIR)/azure.sh nodepool up -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh test ctr -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh nodepool delete -y $(NODEPOOL) )
