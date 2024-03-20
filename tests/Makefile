.PHONY: tests-build
tests-build: ## Builds the tests binary
	@echo "+ $@"
	@( cd $(ROOT_DIR); $(GOBUILD) -o $(TESTS_BIN_DIR)/tests ./tests/cmd )

.PHONY: tests-scanner-image
tests-scanner-image: ## Builds the 'scanner' tests image
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := localhost)
endif
	$(call build-image-internal,$(ROOT_DIR)/tests/dockerfiles/scanner.Dockerfile,scanner,$(ROOT_DIR))

.PHONY: tests-random-image
tests-random-image: ## Builds the 'random' tests image
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := localhost)
endif
	$(call build-image-internal,$(ROOT_DIR)/tests/dockerfiles/random.Dockerfile,random,$(ROOT_DIR))

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
	@( PEERD_IMAGE_TAG=$(PEERD_IMAGE_TAG) $(ROOT_DIR)/scripts/azure.sh nodepool up -y $(NODEPOOL) )
	@( $(ROOT_DIR)/scripts/azure.sh test ctr -y $(NODEPOOL) )
	@( $(ROOT_DIR)/scripts/azure.sh nodepool delete -y $(NODEPOOL) )
