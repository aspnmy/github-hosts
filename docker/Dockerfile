# GitHub Hosts Resolver - Docker
# 自动解析域名DNS并生成hosts文件，通过HTTP服务暴露

FROM alpine:3.19

RUN apk add --no-cache bind-tools curl netcat-openbsd

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /data

EXPOSE 5000

ENV DOMAINS_URL=https://raw.githubusercontent.com/aspnmy/github-hosts/nokv/domains.txt
ENV INTERVAL=28800
ENV PORT=5000
ENV DNS_SERVER=1.1.1.1

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
