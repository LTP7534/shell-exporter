FROM golang:1.19.10 as builder
COPY . /root
WORKDIR /root
RUN go env -w GO111MODULE=on && \
    go env -w GOPROXY=https://goproxy.cn,direct &&\
    go mod init main.go &&\
    go mod tidy &&\
    GOARCH=arm64 GOOS=linux go build -o exporter main.go

#FROM centos:7
#ENV LANG=en_US.UTF-8
#RUN mkdir -pv /usr/share/app/shell &&\
#    yum -y install epel-release &&\
#    yum -y install jq telnet &&\
#    yum clean all &&\
#    cd /usr/share/app &&\
#    curl -s -O 'https://bootstrap.pypa.io/pip/2.7/get-pip.py' &&\
#    python get-pip.py &&\
#    pip install  -i https://mirrors.aliyun.com/pypi/simple/ openpyxl
#WORKDIR /usr/share/app
#COPY --from=builder /root/exporter /usr/share/app/
#CMD "./exporter"
