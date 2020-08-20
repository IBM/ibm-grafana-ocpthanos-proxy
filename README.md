# ibm-grafana-ocpthanos-proxy

> A simple proxy between Grafana and Openshift Container Platform (OCP) thanos-querier service with multi-tenancy enabled

Since OCP 4.3 [Application monitoring](https://docs.openshift.com/container-platform/4.3/monitoring/monitoring-your-own-services.html) is enabled as a technology preview feature. And there is a [thanos-querier](https://github.com/thanos-io/thanos) service as a global query view to enable user to query both cluster metrics and application metrics.

However OCP does not provide Grafana atop of it. This project makes it much easier for user building its own Grafana using the service as its data source. In the meanwhile, it enables namespaces based multi-tenancy support.

## Command line options

- --listen-address
      The address ibm-grafana-ocpthanos-proxy should listen on. Default value: 127.0.0.1:9096
- --url-prefix
  url prefix of the proxy. Default value is "/"
- --thanos-address
  The address of thanos-querier service. Default value: `https://thanos-querier.openshift-monitoring.svc:9091`
- --ns-parser-conf
  NSParser configurate file location. Default value: "/etc/conf/ns-config.yaml"
- --thanos-token-file
  The token file passed to OCP thanos-querier service for authentication. Default value: "/var/run/secrets/kubernetes.io/serviceaccount/token"
- --ns-label-name
  The name of metrics' namespace label. Defalut value: namespace

## Getting Started

The example below can not be used in production environment. It production environment it should be used as sidecar of Grafana Pod and listen to loopback interface only.
1. Enable OCP Application monitoring according to [OCP official document](https://docs.openshift.com/container-platform/4.3/monitoring/monitoring-your-own-services.html). If you just want to play with cluster metrics this step can be ignored.
2. Set up this proxy as a standalone OCP service.
   It can be done by `oc create -f example/openshift.yaml`. The default configuration assuming you have [IBM Bedrock Services](https://github.com/IBM/ibm-common-service-operator) installed and it uses its IAM service as namespace provider.
   If you do not have IBM Bedrock Services installed, edit example/openshift.yaml to update its thanos-proxy-ns-config configmap to use another namespace provider.
3. Install Grafana into your cluster. You can install it via [IBM Bedrock Grafana service](https://github.com/IBM/ibm-monitoring-grafana-operator) or the community Grafana operator from OCP OperatorHub.
4. Configure thanos-proxy as datasource of Grafana
   - click `Add data source button` on grafana's datasources configuration page and select `Promethues` datasource type.
   - Name the datasource name as `thanos`
   - use `http://thanos-proxy:9096` as HTTP URL
   - add `cfc-access-token-cookie` into Whitelisted Cookies if you are using IBM Bedrock Grafana.
5. Now you are ready to create Grafana dashboard using thanos as its datasource.

## Limitations

1. The proxy does not provide TLS encryption and any authentication/authorization. It is expected to be used as sidecar of Grafana pod and listen to loopback interface only.
1. The proxy use namespace label as matcher for multi-tenancy. So the query in Grafana should only use following format matcher for namespace label.
    - No namespace matcher at all.
     The query will be updated to `metric_name{namespace=~"namespace1|namespace2"}`
    -  Use Equal operator only. `{namespace="namespace1"}` for example.
     If `namespace1` is allowed by NSParser, query will be passed onto thanos service without change. Otherwise empty data will be returned.
    -  Use simple `=~` operator. `{namespace=~"namespace1|namespace2"}` for example.
     Query will be passed onto thanos service only if both namespace1 and namespace2 are allowed by NSParser. Otherwise empty data will be returned.


