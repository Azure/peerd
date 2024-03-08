.PHONY: tests-deps-install
tests-deps-install: ## Install dependencies for testing (supported only on Ubuntu)
	@echo "+ $@"
	sudo $(ROOT_DIR)/build/ci/scripts/install-deps.sh

.PHONY: kind-create
kind-create: ## Creates a kind cluster
	@echo "+ $@"
	@( $(ROOT_DIR)/build/ci/scripts/kind.sh cluster create -y )

.PHONY: kind-get
kind-get: ## Shows the current kind cluster
	@echo "+ $@"
	@( $(ROOT_DIR)/build/ci/scripts/kind.sh cluster get -y )

.PHONY: kind-delete
kind-delete: ## Deletes kind cluster
	@echo "+ $@"
	@( $(ROOT_DIR)/build/ci/scripts/kind.sh cluster delete -y )

.PHONY: kind-deploy
kind-deploy: ## Deploys the p2p application to kind cluster
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	@( $(ROOT_DIR)/build/ci/scripts/kind.sh app deploy -y $(CONTAINER_REGISTRY)/peerd:dev )

.PHONY: kind-test-ctr
kind-test-ctr: ## Deploys test 'ctr' to the kind cluster
	@echo "+ $@"
	@( $(ROOT_DIR)/build/ci/scripts/kind.sh test ctr -y )

.PHONY: kind-test-random
kind-test-random: ## Deploys test 'random' to the kind cluster
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	@( $(ROOT_DIR)/build/ci/scripts/kind.sh test random -y $(CONTAINER_REGISTRY)/random:dev )
