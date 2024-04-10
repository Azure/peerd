.PHONY: kind-create
kind-create:
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster create -y )

.PHONY: kind-get
kind-get:
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster get -y )

.PHONY: kind-delete
kind-delete:
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster delete -y )

.PHONY: kind-deploy
kind-deploy:
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	@( $(SCRIPTS_DIR)/kind.sh app deploy -y $(CONTAINER_REGISTRY)/peerd:dev )

.PHONY: kind-test-ctr
kind-test-ctr:
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh test ctr -y )

.PHONY: kind-test-random
kind-test-random:
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	@( $(SCRIPTS_DIR)/kind.sh test random -y $(CONTAINER_REGISTRY)/random:dev )

.PHONY: tests-deps-install
tests-deps-install: ## Install dependencies for testing (supported only on Ubuntu)
	@echo "+ $@"
	sudo $(SCRIPTS_DIR)/install-deps.sh

.PHONY: ci-kind-ctr
ci-kind-ctr: build-image kind-delete kind-create kind-deploy kind-test-ctr ## Run the containerd hosts CI in kind.

.PHONY: ci-kind-random
ci-kind-random: build-image tests-random-image kind-delete kind-create kind-deploy kind-test-random ## Run the blobs range CI in kind.

.PHONY: ci-aks-ctr
ci-aks-ctr: ## Run the containerd hosts CI in AKS.
	@echo "+ $@"
ifndef NODEPOOL
	$(eval NODEPOOL := $(shell date +"prel%y%m%d"))
endif
ifndef PEERD_IMAGE_TAG
	$(eval PEERD_IMAGE_TAG := "stable")
endif
	@echo "\033[92mRunning CI NODEPOOL: $(NODEPOOL)\033[0m"
	@( PEERD_IMAGE_TAG=$(PEERD_IMAGE_TAG) PEERD_CONFIGURE_MIRRORS=true $(SCRIPTS_DIR)/azure.sh nodepool up -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh test ctr -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh nodepool delete -y $(NODEPOOL) )

.PHONY: ci-aks-streaming
ci-aks-streaming: ## Run the streaming CI in AKS.
	@echo "+ $@"
ifndef NODEPOOL
	$(eval NODEPOOL := $(shell date +"prels%y%m%d"))
endif
ifndef PEERD_IMAGE_TAG
	$(eval PEERD_IMAGE_TAG := "stable")
endif
	@echo "\033[92mRunning CI NODEPOOL: $(NODEPOOL)\033[0m"
	@( PEERD_IMAGE_TAG=$(PEERD_IMAGE_TAG) PEERD_CONFIGURE_MIRRORS=false $(SCRIPTS_DIR)/azure.sh nodepool up -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh test streaming -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh nodepool delete -y $(NODEPOOL) )