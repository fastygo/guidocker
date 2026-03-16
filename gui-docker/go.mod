module gui-docker

go 1.21

require (
	github.com/a-h/templ v0.0.0
	go.etcd.io/bbolt v1.4.3
	ui8kit v0.0.0
)

replace github.com/a-h/templ => ../localdeps/templ
replace ui8kit => ../ui8kit
replace go.etcd.io/bbolt => ./localdeps/bbolt
