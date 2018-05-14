FROM amazonlinux:latest
USER root

# Install the AWS CLI
RUN yum install -y wget gzip curl tar unzip git libselinux-python xz gcc make libffi-devel openssl-devel sudo python27-pip python27-devel which \
    && pip install -U pip \
    && yum clean all \
    && curl -O https://dl.google.com/go/go1.10.2.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.10.2.linux-amd64.tar.gz \
    && rm -rvf /var/log/* \
    && pip install awscli \
    && chmod a+rw /tmp \
    && mkdir -p /go/src/github.com/tootedom

ENV PATH=${PATH}:/usr/local/go/bin
ENV GOPATH=/go

