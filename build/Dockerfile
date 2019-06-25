FROM alpine:3.6

COPY build/_output/bin/kubedirector /root/kubedirector
RUN chmod +x /root/kubedirector

COPY build/_output/configcli.tgz /root/configcli.tgz

USER root
