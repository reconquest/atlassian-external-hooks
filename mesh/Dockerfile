FROM alpine:3.16

ENV DOCKER_MESH_VERSION=1.3.1

ENV MESH_HOME=/srv/mesh/data

RUN apk update && apk add \
    bash \
    ca-certificates \
    'git<2.37.1' \
    git-daemon \
    openjdk11 \
    && rm -rf /var/cache/apk/*

WORKDIR /srv/mesh
RUN wget https://packages.atlassian.com/mvn/maven-external/com/atlassian/bitbucket/mesh/mesh-distribution/$DOCKER_MESH_VERSION/mesh-distribution-$DOCKER_MESH_VERSION.tar.gz && \
    tar xzvf mesh-distribution-$DOCKER_MESH_VERSION.tar.gz

ENV JAVA_HOME=/usr/lib/jvm/java-11-openjdk/

CMD /srv/mesh/atlassian-bitbucket-mesh-$DOCKER_MESH_VERSION/bin/start-mesh.sh -fg
