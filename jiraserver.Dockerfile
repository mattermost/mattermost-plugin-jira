FROM ubuntu:latest
ARG VERSION=8.22.0
WORKDIR /home
RUN apt update
RUN apt --assume-yes install openjdk-11-jdk curl
RUN mkdir -p downloads
RUN cd downloads
RUN mkdir -p jirahome
RUN curl https://product-downloads.atlassian.com/software/jira/downloads/atlassian-jira-software-${VERSION}.tar.gz --output atlassian-jira-software.tar.gz
RUN mkdir jira && tar xvf atlassian-jira-software.tar.gz -C jira --strip-components 1

ENV JIRA_HOME=/home/downloads/jirahome
CMD ["jira/bin/start-jira.sh", "-fg"]
