# can use Local.mk to override the image var
-include Local.mk

SHELL = bash

default_image := bluek8s/kubedirector:unstable
image ?= ${default_image}

cluster_resource_name := KubeDirectorCluster
app_resource_name := KubeDirectorApp

bin_dir := /usr/local/bin
project_name := kubedirector

configcli_dir := /root
configcli_version := 0.5

local_deploy_yaml := deploy/kubedirector/deployment-localbuilt.yaml

UNAME := $(shell uname)

ifeq ($(UNAME), Linux)
sedseparator =
sedignorecase = 'I'
else
# macOS sed syntax
sedseparator = ''
sedignorecase =
endif

build_dir := build/_output
configcli_dest := $(build_dir)/configcli.tgz
goarch := amd64
cgo_enabled := 0 

.DEFAULT_GOAL := build

build: configcli pkg/apis/kubedirector.bluedata.io/v1alpha1/zz_generated.deepcopy.go | $(build_dir)
	@echo \* Creating KubeDirector deployment image and YAML...
	@test -d vendor || dep ensure -v
	operator-sdk build ${image}
	@docker image prune -f > /dev/null
	@sed ${sedseparator} -e 's~REPLACE_IMAGE~${image}~' deploy/operator.yaml >${local_deploy_yaml}
	@echo done
	@echo

debug: export GOFLAGS += -gcflags=all=-N -gcflags=-l
debug: build
	@echo \* Applying debug settings to KubeDirector deployment YAML...
	@sed -i ${sedseparator} \
        -e '\~${bin_dir}/${project_name}~s~~${bin_dir}/dlv --listen=:40000 --headless --api-version=2 exec &~' \
        -e '\~template:~,\~metadata:~ {' \
        -e '\~metadata:~ {' \
        -e 'a\      annotations:' \
        -e 'a\        container.apparmor.security.beta.kubernetes.io/${project_name}: unconfined' \
        -e '}' \
        -e '}' \
        -e '\~image:~ {' \
        -e 'a\          ports:' \
        -e 'a\          - containerPort: 40000' \
        -e 'a\            hostPort: 40000' \
        -e '}' \
        -e '$$ {' \
        -e 'a\          securityContext:' \
        -e 'a\            capabilities:' \
        -e 'a\              add:' \
        -e 'a\                - SYS_PTRACE' \
        -e '}' \
        ${local_deploy_yaml}
	@echo done
	@echo

configcli:  | $(build_dir)
	@if [ -e $(configcli_dest) ]; then exit 0; fi;                             \
     echo "\* Downloading configcli package ...";                              \
     curl -L -o $(configcli_dest) https://github.com/bluek8s/configcli/archive/v$(configcli_version).tar.gz

pkg/apis/kubedirector.bluedata.io/v1alpha1/zz_generated.deepcopy.go:  \
        pkg/apis/kubedirector.bluedata.io/v1alpha1/kubedirectorapp_types.go \
        pkg/apis/kubedirector.bluedata.io/v1alpha1/kubedirectorcluster_types.go \
        pkg/apis/kubedirector.bluedata.io/v1alpha1/kubedirectorconfig_types.go
	@test -d vendor || dep ensure -v
	operator-sdk generate k8s

push:
	@set -e; \
        if [[ "${image}" == "${default_image}" ]]; then \
            if [[ "${push_default}" == "" ]]; then \
                echo "Use Local.mk to set the image variable, rebuild, then push."; \
                exit 0; \
            fi; \
        fi; \
        echo docker push ${image}; \
        docker push ${image}
	@echo

deploy:
	@set -e; \
        pods_gone=False; \
        kubectl get -o jsonpath='{.items[0].metadata.name}' pods -l name=${project_name} &> /dev/null || pods_gone=True; \
        if [[ "$$pods_gone" != "True" ]]; then \
            echo "KubeDirector pod already exists. Maybe old pod is still terminating?"; \
            exit 1; \
        fi; \
        kubectl_ns=`kubectl config get-contexts | grep '^\*' | awk '{print $$5}'`; \
        if [[ -z "$$kubectl_ns" ]]; then \
            cp -f deploy/kubedirector/rbac-default.yaml deploy/kubedirector/rbac.yaml; \
        else \
            sed "s/namespace: default/namespace: $$kubectl_ns/" deploy/kubedirector/rbac-default.yaml > deploy/kubedirector/rbac.yaml; \
        fi
	@echo
	@echo \* Creating service account...
	kubectl create -f deploy/kubedirector/rbac.yaml
	@echo
	@echo \* Creating custom resource definitions...
	kubectl create -f deploy/kubedirector/kubedirector_v1alpha1_kubedirectorapp_crd.yaml
	kubectl create -f deploy/kubedirector/kubedirector_v1alpha1_kubedirectorcluster_crd.yaml
	kubectl create -f deploy/kubedirector/kubedirector_v1alpha1_kubedirectorconfig_crd.yaml
	@echo
	@set -e; \
        if [[ -f deploy/kubedirector/deployment-localbuilt.yaml ]]; then \
            echo \* Deploying KubeDirector...; \
            kubectl create -f deploy/kubedirector/deployment-localbuilt.yaml; \
            echo kubectl create -f deploy/kubedirector/deployment-localbuilt.yaml; \
        else \
            echo \* Deploying PRE-BUILT KubeDirector...; \
            echo kubectl create -f deploy/kubedirector/deployment-prebuilt.yaml; \
            kubectl create -f deploy/kubedirector/deployment-prebuilt.yaml; \
        fi; \
        echo; \
        echo -n \* Waiting for KubeDirector to start...; \
        sleep 3; \
        retries=0; \
        while true; do \
            if kubectl get pods -l name=${project_name} &> /dev/null; then \
                break; \
            else \
                retries=`expr $$retries + 1`; \
                if [ $$retries -gt 20 ]; then \
                    echo; \
                    echo KubeDirector failed to start -- no pod created!; \
                    exit 1; \
                fi; \
                echo -n .; \
                sleep 3; \
            fi; \
        done; \
        retries=0; \
        while true; do \
            if kubectl get MutatingWebhookConfiguration ${project_name}-webhook &> /dev/null; then \
                exit 0; \
            else \
                retries=`expr $$retries + 1`; \
                if [ $$retries -gt 20 ]; then \
                    echo; \
                    echo KubeDirector failed to start -- no admission control hook created!; \
                    exit 1; \
                fi; \
                echo -n .; \
                sleep 3; \
            fi; \
        done
	@echo
	@echo
	@echo \* Creating example application types...
	kubectl create -f deploy/example_catalog/
	@echo
	@set -e; \
        podname=`kubectl get -o jsonpath='{.items[0].metadata.name}' pods -l name=${project_name}`; \
        echo KubeDirector pod name is $$podname
	@echo

redeploy:
	@echo
	@echo \* Killing current KubeDirector process \(if any\)...
	@set -e; \
        podname=`kubectl get -o jsonpath='{.items[0].metadata.name}' pods -l name=${project_name}`; \
        kubectl exec $$podname -- killall ${project_name} || true
	@echo
	@echo \* Injecting new configcli package...
	@set -e; \
        podname=`kubectl get -o jsonpath='{.items[0].metadata.name}' pods -l name=${project_name}`; \
        kubectl exec $$podname -- mv -f ${configcli_dir}/configcli.tgz ${configcli_dir}/configcli.tgz.bak || true; \
        kubectl cp ${build_dir}/configcli.tgz $$podname:${configcli_dir}/configcli.tgz
	@echo
	@echo \* Injecting and starting new KubeDirector binary...
	@set -e; \
        podname=`kubectl get -o jsonpath='{.items[0].metadata.name}' pods -l name=${project_name}`; \
        kubectl exec $$podname -- /bin/sh -c "echo REDEPLOYING > /tmp/fifo"; \
        kubectl exec $$podname -- mv -f ${bin_dir}/${project_name} ${bin_dir}/${project_name}.bak || true; \
        kubectl cp ${build_dir}/bin/${project_name} $$podname:${bin_dir}/${project_name}; \
        kubectl exec $$podname -- chmod +x ${bin_dir}/${project_name}; \
        kubectl exec -t $$podname -- /bin/sh -c "${bin_dir}/${project_name} &> /tmp/fifo &"; \
        echo; \
        echo KubeDirector pod name is $$podname
	@echo

undeploy:
	@echo
	@set -e; \
		all_namespaces=`kubectl get ns --no-headers| awk '{print $$1}'`; \
		echo Deleting any managed virtual clusters...; \
		for ns in $$all_namespaces; do \
			echo kubectl -n $$ns delete ${cluster_resource_name} --all --now; \
			kubectl -n $$ns delete ${cluster_resource_name} --all --now || true; \
		done; \
		echo Deleting application types...; \
		for ns in $$all_namespaces; do \
			echo kubectl -n $$ns delete ${app_resource_name} --all --now; \
			kubectl -n $$ns delete ${app_resource_name} --all --now || true; \
		done;
	@echo
	@echo \* Deleting KubeDirector deployment...
	-@if [[ -f deploy/kubedirector/deployment-localbuilt.yaml ]]; then \
        echo kubectl delete -f deploy/kubedirector/deployment-localbuilt.yaml --now; \
        kubectl delete -f deploy/kubedirector/deployment-localbuilt.yaml --now; \
    else \
        echo kubectl delete -f deploy/kubedirector/deployment-prebuilt.yaml --now; \
        kubectl delete -f deploy/kubedirector/deployment-prebuilt.yaml --now; \
    fi
	@echo
	@echo \* Deleting custom resource definitions...
	-kubectl delete -f deploy/kubedirector/kubedirector_v1alpha1_kubedirectorapp_crd.yaml --now
	-kubectl delete -f deploy/kubedirector/kubedirector_v1alpha1_kubedirectorcluster_crd.yaml --now
	-kubectl delete -f deploy/kubedirector/kubedirector_v1alpha1_kubedirectorconfig_crd.yaml --now
	@echo
	@echo \* Deleting service account...
	-@if [[ -f deploy/kubedirector/rbac.yaml ]]; then \
        echo kubectl delete -f deploy/kubedirector/rbac.yaml --now; \
        kubectl delete -f deploy/kubedirector/rbac.yaml --now; \
    else \
        echo kubectl delete -f deploy/kubedirector/rbac-default.yaml --now; \
        kubectl delete -f deploy/kubedirector/rbac-default.yaml --now; \
    fi
	@echo
	@echo -n \* Waiting for all cluster resources to finish cleanup...
	@set -e; \
        retries=100; \
        while [ $$retries ]; do \
            if kubectl get all -l kubedirectorcluster --all-namespaces 2>&1 >/dev/null | grep "No resources found." &> /dev/null; then \
                exit 0; \
            else \
                retries=`expr $$retries - 1`; \
                if [ $$retries -le 0 ]; then \
                    echo; \
                    echo Some KubeDirector-managed resources seem to remain.; \
                    echo Use "kubectl get all -l kubedirectorcluster --all-namespaces" to check and do manual cleanup.; \
                    exit 1; \
                fi; \
                sleep 3; \
                echo -n .; \
            fi; \
        done
	@echo
	@echo
	@echo \* Deleting any storage class labelled kubedirector-support...
	-kubectl delete storageclass -l kubedirector-support=true
	@echo
	@echo done
	@echo

teardown: undeploy

compile: clean
	GOARCH=${goarch} CGO_ENABLED=${cgo_enabled} \
		go build -o ${build_dir}/bin/${project_name} ./cmd/manager

format:
	go fmt $(shell go list ./... | grep -v /vendor/)

dep:
	dep ensure -v -update

clean:
	-rm -f deploy/kubedirector/rbac.yaml
	-rm -f deploy/kubedirector/deployment-localbuilt.yaml
	-rm -rf ${build_dir}

distclean: clean
	-rm -rf vendor

modules:
	GO111MODULE="on" go mod tidy

verify-modules:
	rm -f go.mod go.sum
	-GO111MODULE="on" go mod init
	-GO111MODULE="on" go mod tidy
	@# This line checks that we haven't changed the go.mod or go.sum file
	@# apart from the first line (because Travis thinks that the local build
	@# is under the _user's own_ module)
	@if [ $$(git --no-pager diff --unified=0 --no-color -- go.mod go.sum | \
             grep -Ev "^(-{3}|\+{3}|\@{2}|diff|index).*$$" | \
             grep -Ev ".*github.com/.+?/kubedirector.*$$" | \
             wc -c) -eq 0 ] ; then \
        echo "no module changes, good job!" ; \
    else \
        echo "changes to go modules" ; \
        echo "make sure to run \`make modules\` before checking in" ; \
        git --no-pager diff --unified=0 -- go.mod go.sum ; \
        dep version ; \
        exit 1 ; \
    fi

golint:
	@if [ $$(golint \
            $$(go list -f '{{.Dir}}' ./...) | \
        grep -v "generated.deepcopy.go:" | \
        wc -l) -eq 0 ] ; then \
        echo "No new golint issues, good job!" ; \
    else \
        echo "There were some new golint issues:" ; \
        golint_out=$$(golint \
            $$(go list -f '{{.Dir}}' ./...) | \
        grep -v "generated.deepcopy.go:") ; \
        echo $$golint_out ; \
        exit 1 ; \
    fi

check-format:
	@make clean > /dev/null
	@if [ "$$(gofmt -d $$(go list -f '{{.Dir}}' ./...))" == "" ] ; then \
	    echo "No formatting changes needed, good job!" ; \
    else \
	    echo "Formatting changes necessary, please run make format and resubmit" ; \
	    echo "$$(gofmt -d $$(go list -f '{{.Dir}}' ./...))" ; \
        exit 2 ; \
    fi


$(build_dir):
	@mkdir -p $@

.PHONY: build push deploy redeploy undeploy teardown format dep clean distclean compile verify-modules modules golint check-format
