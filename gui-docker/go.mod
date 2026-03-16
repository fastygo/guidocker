module gui-docker

go 1.23.0

require (
	github.com/a-h/templ v0.3.1001
	go.etcd.io/bbolt v1.4.3
	ui8kit v0.0.0
)

replace ui8kit => ../ui8kit

replace go.etcd.io/bbolt => ./localdeps/bbolt
