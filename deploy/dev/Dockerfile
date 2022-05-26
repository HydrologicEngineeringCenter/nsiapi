FROM golang:1.18.2-buster

ENV TZ=America/New_York

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone &&\
    apt update &&\
    apt -y install gdal-bin gdal-data libgdal-dev
