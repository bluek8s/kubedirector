# can use Local.mk to override the image var
-include Local.mk

SHELL = bash

default_image := bluek8s/kubedirector:unstable
image ?= ${default_image}

cluster_resource_name := kubedirectorcluster
app_resource_name := kubedirectorapp
config_resource_name := kubedirectorconfig

project_name := kubedirector
bin_name := kubedirector

home_dir := /home/kubedirector
configcli_version := 0.7.2

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
configcli_dest := build/configcli.tgz
goarch := amd64
cgo_enabled := 0

.DEFAULT_GOAL := build

version-check:
	@if go version | grep -q 'go1\.1[3-9]'; then \
        true; \
    else \
        echo "Error:"; \
        echo "go version 1.13 or later is required"; \
        exit 1; \
    fi
	@if operator-sdk version | grep -q 'operator-sdk version: "v0.15.2'; then \
        true; \
    else \
        echo "Error:"; \
        echo "operator-sdk version 0.15.2 is required"; \
        exit 1; \
    fi

build: configcli pkg/apis/kubedirector/v1beta1/zz_generated.deepcopy.go version-check | $(build_dir)
	@echo
	@echo \* Creating KubeDirector deployment image and YAML...
	operator-sdk build ${image}
	@docker image prune -f > /dev/null
	@sed -e 's~REPLACE_IMAGE~${image}~' deploy/operator.yaml >${local_deploy_yaml}
	@echo done
	@echo

configcli:
	@if [ -e $(configcli_dest) ]; then exit 0; fi;                             \
     echo "* Downloading configcli package ...";                               \
     curl -L -o $(configcli_dest) https://github.com/bluek8s/configcli/archive/v$(configcli_version).tar.gz

pkg/apis/kubedirector/v1beta1/zz_generated.deepcopy.go:  \
        pkg/apis/kubedirector/v1beta1/kubedirectorapp_types.go \
        pkg/apis/kubedirector/v1beta1/kubedirectorcluster_types.go \
        pkg/apis/kubedirector/v1beta1/kubedirectorconfig_types.go
	operator-sdk generate k8s

push:
	@set -e; \
        if [[ "${image}" == "${default_image}" ]]; then \
            if [[ "${push_default}" == "" ]]; then \
                echo "Use Local.mk to set the image variable, rebuild, then push."; \
                exit 0; \
            fi; \
        fi
	docker push ${image}
	@echo

deploy:
	@set -e; \
        pods_gone=False; \
        kubectl get -o jsonpath='{.items[0].metadata.name}' pods -l name=${project_name} -A &> /dev/null || pods_gone=True; \
        if [[ "$$pods_gone" != "True" ]]; then \
            echo "KubeDirector pod already exists. Maybe the old pod is still terminating?"; \
            exit 1; \
        fi; \
        kubectl_ns=`kubectl config get-contexts | grep '^\*' | awk '{print $$5}'`; \
        if [[ -z "$$kubectl_ns" ]]; then \
            cp -f deploy/kubedirector/rbac-default.yaml deploy/kubedirector/rbac.yaml; \
        else \
            sed "s/namespace: default/namespace: $$kubectl_ns/" deploy/kubedirector/rbac-default.yaml > deploy/kubedirector/rbac.yaml; \
        fi

	@echo
	@echo \* Creating custom resource definitions...
	kubectl create -f deploy/kubedirector/kubedirector.hpe.com_kubedirectorapps_crd.yaml
	kubectl create -f deploy/kubedirector/kubedirector.hpe.com_kubedirectorclusters_crd.yaml
	kubectl create -f deploy/kubedirector/kubedirector.hpe.com_kubedirectorconfigs_crd.yaml
	@echo
	@echo \* Creating role and service account...
	kubectl create -f deploy/kubedirector/rbac.yaml
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
        kubectl exec $$podname -- mv -f ${home_dir}/configcli.tgz ${home_dir}/configcli.tgz.bak || true; \
        kubectl cp ${configcli_dest} $$podname:${home_dir}/configcli.tgz; \
        kubectl exec $$podname -- chgrp 0 ${home_dir}/configcli.tgz; \
        kubectl exec $$podname -- chmod ug=rw ${home_dir}/configcli.tgz
	@echo
	@echo \* Injecting and starting new KubeDirector binary...
	@set -e; \
        podname=`kubectl get -o jsonpath='{.items[0].metadata.name}' pods -l name=${project_name}`; \
        kubectl exec $$podname -- /bin/sh -c "echo REDEPLOYING > /tmp/fifo"; \
        kubectl exec $$podname -- mv -f ${home_dir}/${project_name} ${home_dir}/${project_name}.bak || true; \
        kubectl cp ${build_dir}/bin/${bin_name} $$podname:${home_dir}/${project_name}; \
        kubectl exec $$podname -- chgrp 0 ${home_dir}/${project_name}; \
        kubectl exec $$podname -- chmod ug=rwx ${home_dir}/${project_name}; \
        kubectl exec -t $$podname -- /bin/sh -c "${home_dir}/${project_name} &> /tmp/fifo &"; \
        echo; \
        echo KubeDirector pod name is $$podname
	@echo

undeploy:
	@echo
	@true; \
        function delete_cluster_thing { \
            kind=$$1; \
            name=$$2; \
            cmd="kubectl delete $$kind $$name --now"; \
            msg=$$($$cmd 2>&1); \
            if [[ "$$?" == "0" ]]; then \
                echo $$cmd; \
                if [[ "$$msg" != "" ]]; then \
                    echo "$$msg"; \
                fi; \
            else \
                if [[ ! "$$msg" =~ "Error from server (NotFound):" ]]; then \
                    echo $$cmd; \
                    echo "$$msg"; \
                    exit 1; \
                fi; \
            fi; \
        }; \
        function delete_namespaced_thing { \
            kind=$$1; \
            name=$$2; \
            ns_s_containing_kd_cmd="kubectl get $$kind -A --field-selector=$"metadata.name=$$name$" -o jsonpath='{.items[*].metadata.namespace}'"; \
            ns_s_containing_kd=$$($$ns_s_containing_kd_cmd); \
            for ns in $$ns_s_containing_kd; do \
                ns=$$(echo "$$ns" | tr -d "'"); \
                cmd="kubectl delete $$kind $$name -n $$ns --now"; \
                msg=$$($$cmd 2>&1); \
                if [[ "$$?" == "0" ]]; then \
                    echo $$cmd; \
                    if [[ "$$msg" != "" ]]; then \
                        echo "$$msg"; \
                    fi; \
                else \
                    if [[ ! "$$msg" =~ "Error from server (NotFound):" ]]; then \
                        echo $$cmd; \
                        echo "$$msg"; \
                        exit 1; \
                    fi; \
                fi; \
            done; \
        }; \
        function delete_all_things { \
            cmd="kubectl delete $$1 --all=true -A --now"; \
            msg=$$($$cmd 2>&1); \
            if [[ "$$?" == "0" ]]; then \
                if [[ "$$msg" != "No resources found" ]]; then \
                    echo $$cmd; \
                    if [[ "$$msg" != "" ]]; then \
                        echo "$$msg"; \
                    fi; \
                fi; \
            else \
                if [[ ! "$$msg" =~ "the server doesn't have a resource type" ]]; then \
                    echo $$cmd; \
                    echo "$$msg"; \
                    exit 1; \
                fi; \
            fi; \
        }; \
        echo \* Deleting any managed virtual clusters...; \
        delete_all_things ${cluster_resource_name}; \
        echo; \
        echo \* Deleting any application types...; \
        delete_all_things ${app_resource_name}; \
        echo; \
        echo \* Deleting any configs...; \
        delete_all_things ${config_resource_name}; \
        echo; \
        echo \* Deleting KubeDirector deployment...; \
        delete_namespaced_thing deployment ${project_name}; \
        echo; \
        echo \* Deleting role and service account...; \
        delete_cluster_thing clusterrolebinding ${project_name}; \
        delete_cluster_thing clusterrole ${project_name}; \
        delete_namespaced_thing serviceaccount ${project_name}; \
        echo; \
        echo \* Deleting custom resource definitions...; \
        delete_cluster_thing customresourcedefinition ${app_resource_name}s.kubedirector.hpe.com; \
        delete_cluster_thing customresourcedefinition ${cluster_resource_name}s.kubedirector.hpe.com; \
        delete_cluster_thing customresourcedefinition ${config_resource_name}s.kubedirector.hpe.com
	@echo
	@echo -n \* Waiting for all cluster resources to finish cleanup...
	@set -e; \
        retries=100; \
        while [ $$retries ]; do \
            if kubectl get all -l kubedirector.hpe.com/kdcluster --all-namespaces 2>&1 >/dev/null | grep "No resources found" &> /dev/null; then \
                exit 0; \
            else \
                retries=`expr $$retries - 1`; \
                if [ $$retries -le 0 ]; then \
                    echo; \
                    echo Some KubeDirector-managed resources seem to remain.; \
                    echo Use "kubectl get all -l kubedirector.hpe.com/kdcluster --all-namespaces" to check and do manual cleanup.; \
                    exit 1; \
                fi; \
                sleep 3; \
                echo -n .; \
            fi; \
        done
	@echo
	@echo
	@echo \* Deleting any storage class labelled kubedirector-support...
	@true; \
        cmd="kubectl delete storageclass -l kubedirector-support=true --now"; \
        msg=$$($$cmd 2>&1); \
        if [[ "$$?" == "0" ]]; then \
            if [[ "$$msg" != "No resources found" ]]; then \
                echo $$cmd; \
                if [[ "$$msg" != "" ]]; then \
                    echo "$$msg"; \
                fi; \
            fi; \
        else \
            echo $$cmd; \
            echo "$$msg"; \
            exit 1; \
        fi
	@echo
	@echo done
	@echo

teardown: undeploy

compile: version-check configcli pkg/apis/kubedirector/v1beta1/zz_generated.deepcopy.go
	-rm -rf ${build_dir}
	GOOS=linux GOARCH=${goarch} CGO_ENABLED=${cgo_enabled} \
        go build -gcflags "all=-trimpath=$$GOPATH" -o ${build_dir}/bin/${bin_name} ./cmd/manager

format:
	go fmt $(shell go list ./...)

clean:
	-rm -f deploy/kubedirector/rbac.yaml
	-rm -f deploy/kubedirector/deployment-localbuilt.yaml
	-rm -f pkg/apis/kubedirector/v1beta1/zz_generated.deepcopy.go
	-rm -rf ${build_dir}
	-rm -f ${configcli_dest}

modules:
	go mod tidy

tidy: modules

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

.PHONY: version-check build configcli push deploy redeploy undeploy teardown compile format clean modules tidy golint check-format
