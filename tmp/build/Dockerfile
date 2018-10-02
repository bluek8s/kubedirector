FROM alpine:3.6

ADD tmp/_output/bin/kubedirector /root/kubedirector
RUN chmod +x /root/kubedirector

COPY tmp/_output/nodeprep.tgz /root/nodeprep.tgz

USER root
