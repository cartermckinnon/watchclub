VERSION 0.8
ARG --global IMAGE_REPO="ghcr.io/cartermckinnon"

proto-builder:
    # toolchain last updated: April 16, 2022.
    FROM ubuntu:26.04
    # Get rid of the warning: "debconf: unable to initialize frontend: Dialog"
    # https://github.com/moby/moby/issues/27988
    RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections
    RUN apt-get update && apt-get install wget unzip golang git npm -y
    # https://github.com/protocolbuffers/protobuf/releases
    WORKDIR /tmp
    RUN wget -O protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.20.0/protoc-3.20.0-linux-x86_64.zip && \
        unzip protoc.zip -d /protoc
    ENV PATH=$PATH:/protoc/bin
    # https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go?tab=versions
    RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0
    # https://pkg.go.dev/google.golang.org/grpc/cmd/protoc-gen-go-grpc
    RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
    # Install grpc-web plugin
    RUN wget -O /usr/local/bin/protoc-gen-grpc-web https://github.com/grpc/grpc-web/releases/download/1.4.2/protoc-gen-grpc-web-1.4.2-linux-x86_64 && \
        chmod +x /usr/local/bin/protoc-gen-grpc-web
    ENV PATH=$PATH:/root/go/bin

proto:
    FROM +proto-builder
    WORKDIR /proto
    COPY proto/*.proto .
    RUN mkdir go/ js/
    RUN /protoc/bin/protoc \
        -I=/proto \
        --go_out=go/ \
        --go-grpc_out=go/ \
        --js_out=import_style=commonjs:js/ \
        --grpc-web_out=import_style=commonjs,mode=grpcwebtext:js/ \
        *.proto
    # disable eslint on generated JS files (https://github.com/grpc/grpc-web/issues/447)
    RUN find js/ -type f -exec sh -c "echo '/* eslint-disable */' | cat - {} > /tmp/out && mv /tmp/out {}" \;
    SAVE ARTIFACT go/internal/api/ /go AS LOCAL internal/api
    SAVE ARTIFACT js/ /js AS LOCAL ui/src/api

builder:
    FROM golang
    WORKDIR /go/src/github.com/cartermckinnon/watchclub
    COPY . .
    COPY +proto/go internal/api
    ARG GIT_COMMIT="unknown"
    ARG VERSION="dev"
    RUN go build \
        -ldflags "-X github.com/cartermckinnon/watchclub/internal/version.GitCommit=${GIT_COMMIT} \
                  -X github.com/cartermckinnon/watchclub/internal/version.Version=${VERSION}" \
        -o /go/bin/ ./cmd/...
    SAVE ARTIFACT /go/bin/watchclub /watchclub AS LOCAL bin/watchclub

watchclub:
    FROM ubuntu:26.04
    RUN apt-get update && apt-get install -y ca-certificates
    LABEL org.opencontainers.image.source="https://github.com/cartermckinnon/watchclub/"
    ARG GIT_COMMIT="unknown"
    ARG VERSION="latest"
    COPY (+builder/watchclub --GIT_COMMIT=${GIT_COMMIT} --VERSION=${VERSION}) /usr/bin/watchclub
    SAVE ARTIFACT /usr/bin/watchclub /watchclub AS LOCAL bin/watchclub
    ENTRYPOINT ["/usr/bin/watchclub"]
    CMD ["server"]
    SAVE IMAGE --push $IMAGE_REPO/watchclub:$VERSION

ui-builder:
    FROM node:lts
    WORKDIR /workdir
    COPY ui/package.json .
    COPY ui/package-lock.json .
    COPY ui/webpack.config.js .
    RUN npm install
    COPY ui/src src/
    COPY +proto/js src/api
    RUN npm run build && \
        mkdir -p build/css/ && \
        cp src/css/* build/css/
    SAVE ARTIFACT /workdir/build /ui

ui:
    FROM nginx:stable
    LABEL org.opencontainers.image.source="https://github.com/cartermckinnon/watchclub"
    COPY +ui-builder/ui /var/www
    COPY ui/nginx.conf /etc/nginx/conf.d/default.conf
    CMD ["nginx","-g","daemon off;"]
    ARG VERSION="latest"
    SAVE IMAGE --push $IMAGE_REPO/watchclub/ui:$VERSION

crane-builder:
    FROM golang
    RUN go install github.com/google/go-containerregistry/cmd/crane@latest
    SAVE ARTIFACT /go/bin/crane /crane

image-updater:
    FROM ubuntu:26.04
    LABEL org.opencontainers.image.source="https://github.com/cartermckinnon/watchclub"
    COPY +crane-builder/crane /usr/bin/crane
    COPY ops/update-image.sh /usr/bin/update-image.sh
    RUN apt-get update && \
      apt-get install -y ca-certificates curl && \
      curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
      chmod +x kubectl && \
      mv kubectl /usr/bin/kubectl
    ARG VERSION="latest"
    SAVE IMAGE --push $IMAGE_REPO/watchclub/image-updater:$VERSION
