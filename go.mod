module github.com/bluek8s/kubedirector

go 1.12

require (
	cloud.google.com/go v0.39.0 // indirect
	github.com/Azure/azure-sdk-for-go v31.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.5.0 // indirect
	github.com/NYTimes/gziphandler v1.0.1 // indirect
	github.com/beorn7/perks v1.0.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.2
	github.com/google/uuid v1.1.1
	github.com/gophercloud/gophercloud v0.2.0 // indirect
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c // indirect
	github.com/kisielk/errcheck v1.2.0 // indirect
	github.com/operator-framework/operator-sdk v0.9.0
	github.com/prometheus/common v0.4.0 // indirect
	github.com/prometheus/procfs v0.0.0-20190516134534-5de912679dde // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/pflag v1.0.3
	go.uber.org/zap v1.10.0 // indirect
	google.golang.org/appengine v1.6.0 // indirect
	k8s.io/api v0.0.0-20190612125737-db0771252981
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.0.0-20190726022633-14ba7d03f06f // indirect
	k8s.io/kube-openapi v0.0.0-20190709113604-33be087ad058
	sigs.k8s.io/controller-runtime v0.1.12
)

// see https://github.com/azure/go-autorest#using-with-go-modules
replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.3.0+incompatible
