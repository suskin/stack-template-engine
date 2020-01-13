
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

GO111MODULE ?= on
export GO111MODULE

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

helpers:
	docker build . -f helm.Dockerfile --tag 'crossplane/helm-engine:latest'
	docker build . -f kubectl.Dockerfile --tag 'crossplane/kubectl:latest'

.PHONY: helpers

integration-test: helpers integration-test-helm2
.PHONY: integration-test

integration-test-helm2:
	docker build test/helm2 --tag 'crossplane/sample-stack-claim-test:helm2'
	kubectl apply -f test/helm2/sample-crd.yaml
	kubectl apply -f test/helm2/stack.yaml
	kubectl apply -f test/helm2/sample-cr.yaml
	@echo "Printing test object statuses"
	kubectl get job -A
	kubectl get pod -A
	@echo "Giving the controller some time to process our resources . . ."
	sleep 10
	@echo "If the config map 'mycustomname' isn't found, try looking for it again, or inspect the job logs to debug."
	kubectl get configmap mycustomname-helm2 -o yaml

.PHONY: integration-test-helm2

clean-integration-test: clean-integration-test-helm2

.PHONY: clean-integration-test

clean-integration-test-helm2:
	kubectl delete configmap mycustomname
	kubectl delete -f test/helm2/sample-cr.yaml
	kubectl delete -f test/helm2/stack.yaml
	kubectl delete -f test/helm2/sample-crd.yaml

.PHONY: clean-integration-test-helm2
