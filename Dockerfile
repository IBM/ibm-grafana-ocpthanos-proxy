# Copyright 2020 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#Always get the latest
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.2-349

ARG IMAGE_NAME
ARG IMAGE_DISPLAY_NAME
ARG IMAGE_NAME_ARCH
ARG IMAGE_MAINTAINER
ARG IMAGE_VENDOR
ARG IMAGE_VERSION
ARG IMAGE_RELEASE
ARG IMAGE_DESCRIPTION
ARG IMAGE_SUMMARY
ARG IMAGE_OPENSHIFT_TAGS
ARG VCS_REF
ARG VCS_URL

LABEL org.label-schema.vendor=$IMAGE_VENDOR \
  org.label-schema.name=$IMAGE_NAME_ARCH \
  org.label-schema.description=$IMAGE_DESCRIPTION \
  org.label-schema.vcs-ref=$VCS_REF \
  org.label-schema.vcs-url=$VCS_URL \
  org.label-schema.license="Licensed Materials - Property of IBM" \
  org.label-schema.schema-version=$IMAGE_VERSION \
  name=$IMAGE_NAME \
  vendor=$IMAGE_VENDOR \
  description=$IMAGE_DESCRIPTION \
  maintainer=$IMAGE_MAINTAINER \
  summary=$IMAGE_DESCRIPTION

COPY grafana-ocpthanos-proxy /usr/local/bin/grafana-ocpthanos-proxy

# copy licenses
RUN mkdir /licenses && microdnf update
COPY LICENSE /licenses

ENTRYPOINT ["grafana-ocpthanos-proxy"]
USER 66666
