module github.com/suskin/stack-template-engine

go 1.12

require (
	github.com/crossplaneio/crossplane-runtime v0.2.3
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/google/btree v1.0.0 // indirect
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/xlab/handysort v0.0.0-20150421192137-fb3537ed64a1 // indirect
	golang.org/x/crypto v0.0.0-20190820162420-60c769a6c586 // indirect
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.0.0-20191121015212-c4c8f8345c7e // indirect
	k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/utils v0.0.0-20191114184206-e782cd3c129f // indirect
	sigs.k8s.io/controller-runtime v0.4.0
	sigs.k8s.io/yaml v1.1.0
	vbom.ml/util v0.0.0-20160121211510-db5cfe13f5cc // indirect
)

replace k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
