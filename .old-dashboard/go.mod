module dashboard

go 1.21

require go.etcd.io/bbolt v1.4.3

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace go.etcd.io/bbolt => ./localdeps/bbolt
