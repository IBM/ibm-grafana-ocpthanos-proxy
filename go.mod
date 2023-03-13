module github.com/IBM/ibm-grafana-ocpthanos-proxy

go 1.18

require (
	github.com/ghodss/yaml v1.0.0
	github.com/prometheus/prometheus v1.8.2-0.20200507164740-ecee9c8abfd1
)

require (
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/go-kit/kit v0.10.0 // indirect
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
)

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

replace golang.org/x/sys => golang.org/x/sys v0.0.0-20220412211240-33da011f77ad
