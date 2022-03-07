FROM ubuntu:latest

ENV TZ=America/New_York
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN set -eux; \
	apt-get update; \
        	apt-get install -y gdal-bin libgdal-dev; \
           mkdir /app; \
           mkdir /app/tmp;

EXPOSE 4141
EXPOSE 5432

COPY . /app/

ENTRYPOINT ["/app/nsiv2-api"]

