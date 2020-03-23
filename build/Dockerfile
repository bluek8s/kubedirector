FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/home/kubedirector/kubedirector \
    USER_UID=1001 \
    USER_NAME=kubedirector \
    USER_HOME=/home/kubedirector

# Support ps and killall, for redeploy.
RUN microdnf update -y && rm -rf /var/cache/yum
RUN microdnf -y install --nodocs psmisc procps-ng \
    && microdnf clean all

COPY build/bin/entrypoint /usr/local/bin/entrypoint
COPY build/bin/user_setup /usr/local/bin/user_setup
RUN chown 0:0 /usr/local/bin/entrypoint && \
    chmod u=rwx,go=rx /usr/local/bin/entrypoint && \
    chown 0:0 /usr/local/bin/user_setup && \
    chmod u=rwx,go=rx /usr/local/bin/user_setup
RUN /usr/local/bin/user_setup

COPY build/_output/bin/kubedirector ${OPERATOR}
COPY build/configcli.tgz ${USER_HOME}/configcli.tgz
RUN chown ${USER_UID}:0 ${OPERATOR} && \
    chmod ug=rwx ${OPERATOR} && \
    chown ${USER_UID}:0 ${USER_HOME}/configcli.tgz && \
    chmod ug=rw ${USER_HOME}/configcli.tgz

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}:0
