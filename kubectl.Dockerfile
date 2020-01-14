FROM alpine:3.11

WORKDIR /tmp

RUN apk add curl bash

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.16.2/bin/linux/amd64/kubectl &&\
    chmod +x kubectl && mv kubectl /usr/local/bin/kubectl

# Kustomize that exists in kubectl is too old.
RUN curl -LO https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz &&\
        tar -xvzf kustomize_v3.5.4_linux_amd64.tar.gz && chmod +x kustomize && mv kustomize /usr/local/bin/kustomize

ENTRYPOINT ["kubectl"]
