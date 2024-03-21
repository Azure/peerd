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
