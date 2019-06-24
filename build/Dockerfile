FROM alpine:3.6

ADD tmp/_output/bin/kubedirector /root/kubedirector
RUN chmod +x /root/kubedirector

COPY tmp/_output/configcli.tgz /root/configcli.tgz

USER root
