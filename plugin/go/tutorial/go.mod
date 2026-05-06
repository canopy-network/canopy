module github.com/canopy-network/go-plugin/tutorial

go 1.24.0

toolchain go1.24.2

require (
	github.com/drand/kyber v1.3.2
	github.com/drand/kyber-bls12381 v0.3.4
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/canopy-network/canopy/plugin/go v0.0.0
	github.com/kilic/bls12-381 v0.1.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
)

replace github.com/canopy-network/canopy/plugin/go => ../.
