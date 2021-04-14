module github.com/DataDog/datadog-agent/pkg/trace/export

go 1.14

replace github.com/DataDog/datadog-agent/pkg/util/log => ../../util/log

require (
	github.com/DataDog/datadog-agent/pkg/util/log v0.0.0
	github.com/DataDog/datadog-go v4.5.1+incompatible
	github.com/DataDog/sketches-go v1.0.0
	github.com/Microsoft/go-winio v0.4.17 // indirect
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575
	github.com/dgraph-io/ristretto v0.0.3
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/shirou/gopsutil v3.21.3+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/tinylib/msgp v1.1.5
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.12
	golang.org/x/sys v0.0.0-20210414055047-fe65e336abe0
	k8s.io/apimachinery v0.21.0
)
