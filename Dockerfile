FROM hub.kce.ksyun.com/wuwen/ubuntu:18.04

WORKDIR /app
COPY bin/wwuc /app
COPY conf/ /app/conf

EXPOSE 8100
EXPOSE 9101

CMD ["/app/wwuc","-conf","/app/conf/wwuc.yaml"]
