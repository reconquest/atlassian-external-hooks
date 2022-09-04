MESH_VERSION = 1.3.1
IMAGE_VERSION = 1
ORG = kovetskiy

build:
	docker build \
		--build-arg "DOCKER_MESH_VERSION=$(MESH_VERSION)" \
		-t $(ORG)/atlassian-bitbucket-mesh:$(MESH_VERSION)-$(IMAGE_VERSION) \
		.

push:
	docker push $(ORG)/atlassian-bitbucket-mesh:$(MESH_VERSION)-$(IMAGE_VERSION)
