#!/bin/echo docker build . -f
# -*- coding: utf-8 -*-
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.12-buster
LABEL maintainer "Philippe Coval (rzr@users.sf.net)"

ENV project kubeedge
ENV project_dir /usr/local/opt/${project}
ENV src_dir ${project_dir}/src/${project}

WORKDIR ${src_dir}
COPY . ${src_dir}/
RUN echo "# log: ${project}: Building sources" \
  && set -x \
  && go version \
  && make \
  && sync
