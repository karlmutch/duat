IMAGE_NAME=$(shell basename $(CURDIR))

.PHONY: docker-lint
docker-lint: ## Run Dockerfile Lint on all dockerfiles.
	for f in */Dockerfile ; do \
	  docker run -it --rm --privileged -v `pwd`:/root/ projectatomic/dockerfile-lint dockerfile_lint -f $f ; \
	done

.PHONY: docker-build
docker-build: ## Build the top level Dockerfile using the directory or $IMAGE_NAME as the name.
	docker build -t $(IMAGE_NAME) .

.PHONY: docker-tag
docker-tag: ## Tag the docker image using the tag script.
	docker tag $(IMAGE_NAME) $(DOCKER_REPO)-$(IMAGE_NAME)

.PHONY: docker-publish
docker-publish: docker-tag ## Publish the image and tags to a repository.
	docker push $(DOCKER_REPO)-$(IMAGE_NAME)
