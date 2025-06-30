.PHONY: build-example-llm-gemma3.27b
build-example-llm-gemma3.27b: ## Build the example LLM image (gemma3:27b).
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	$(call build-image-internal,$(ROOT_DIR)/examples/llm/gemma3.27b.Dockerfile,example-llm,$(ROOT_DIR)/examples/llm)

.PHONY: ci-aks-llm-streaming
ci-aks-llm-streaming: ## Run the artifact streaming based LLM CI in AKS.
	@echo "+ $@"
ifndef NODEPOOL
	$(eval NODEPOOL := $(shell date +"prel%y%m%d"))
endif
ifndef PEERD_IMAGE_TAG
	$(eval PEERD_IMAGE_TAG := "stable")
endif
	@echo "\033[92mRunning CI NODEPOOL: $(NODEPOOL)\033[0m"
	@( PEERD_IMAGE_TAG=$(PEERD_IMAGE_TAG) PEERD_CONFIGURE_MIRRORS=false PEERD_CONFIGURE_OVERLAYBD_P2P=true $(SCRIPTS_DIR)/azure.sh nodepool up -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh peerd_pod_watcher start -y )
	@( $(SCRIPTS_DIR)/azure.sh test llm_streaming -y $(NODEPOOL) )
	@( $(SCRIPTS_DIR)/azure.sh peerd_pod_watcher stop -y )
	@( $(SCRIPTS_DIR)/azure.sh nodepool delete -y $(NODEPOOL) )

.PHONY: ci-kind-random
ci-kind-random: build-image tests-random-image kind-delete kind-create kind-deploy kind-test-random ## Run the blobs range CI in kind.

.PHONY: kind-create
kind-create:
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster create -y )

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

.PHONY: kind-get
kind-get:
	@echo "+ $@"
	@( $(SCRIPTS_DIR)/kind.sh cluster get -y )

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