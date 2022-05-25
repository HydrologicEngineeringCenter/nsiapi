# FROM ubuntu:20.04
FROM ubuntu:22.10

ENV TZ=America/New_York
ENV PATH=/go/bin:$PATH
ENV GOROOT=/go
ENV GOPATH=/src/go

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone &&\
    mkdir /go &&\
    mkdir -p /src/go &&\
    apt update &&\
    apt -y install gdal-bin gdal-data libgdal-dev &&\
    apt -y install wget &&\
    wget https://golang.org/dl/go1.17.10.linux-amd64.tar.gz -P / &&\
    tar -xvzf /go1.17.10.linux-amd64.tar.gz -C / &&\
    apt -y install vim &&\
    apt -y install git
