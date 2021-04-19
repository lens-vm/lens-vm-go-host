module github.com/lens-vm/lens-vm-go-host

go 1.14

require (
	github.com/lens-vm/gogl v0.4.0
	github.com/stretchr/testify v1.7.0
	github.com/wasmerio/wasmer-go v1.0.3
	gopkg.in/fatih/set.v0 v0.2.1 // indirect
)

replace (
	github.com/lens-vm/gogl => ../gogl
)