module github.com/multiversx/mx-chain-vm-go

go 1.23

require (
	github.com/awalterschulze/gographviz v2.0.3+incompatible
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.1
	github.com/gogo/protobuf v1.3.2
	github.com/mitchellh/mapstructure v1.5.0
	github.com/multiversx/mx-chain-core-go v1.4.0
	github.com/multiversx/mx-chain-crypto-go v1.3.0
	github.com/multiversx/mx-chain-logger-go v1.1.0
	github.com/multiversx/mx-chain-scenario-go v1.6.0
	github.com/multiversx/mx-chain-storage-go v1.1.0
	github.com/multiversx/mx-chain-vm-common-go v1.6.0
	github.com/multiversx/mx-components-big-int v1.1.0
	github.com/pelletier/go-toml v1.9.3
	github.com/stretchr/testify v1.10.0
	github.com/urfave/cli/v2 v2.27.1
	golang.org/x/crypto v0.32.0
)

require (
	github.com/TwiN/go-color v1.1.0 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/denisbrodbeck/machineid v1.0.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/golang-lru v0.6.0 // indirect
	github.com/herumi/bls-go-binary v1.28.2 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/sys v0.30.0 // indirect
	google.golang.org/protobuf v1.36.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/multiversx/mx-chain-core-go => github.com/xorewa/mx-chain-core-go v0.0.0-20260622185646-a558edf5ceda

replace github.com/multiversx/mx-chain-crypto-go => github.com/xorewa/mx-chain-crypto-go v0.0.0-20260630162109-4c1e9574b7d4

replace github.com/multiversx/mx-chain-logger-go => github.com/xorewa/mx-chain-logger-go v0.0.0-20260630161826-04c5818227ed

replace github.com/multiversx/mx-chain-storage-go => github.com/xorewa/mx-chain-storage-go v0.0.0-20260630162005-120dc6b184cb

replace github.com/multiversx/mx-chain-vm-common-go => github.com/xorewa/mx-chain-vm-common-go v0.0.0-20260720034041-120a90af8530

replace github.com/multiversx/mx-chain-scenario-go => github.com/xorewa/mx-chain-scenario-go v0.0.0-20260721235302-497db72c941c

replace github.com/multiversx/mx-components-big-int => github.com/xorewa/mx-components-big-int v0.0.0-20260507133911-536f4799b94f

replace github.com/herumi/bls-go-binary => github.com/xorewa/bls-go-binary v0.0.0-20250924002409-446538da6433
