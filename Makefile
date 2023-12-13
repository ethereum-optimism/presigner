ifndef $(GOPATH)
    GOPATH=$(shell go env GOPATH)
    export GOPATH
endif

.PHONY: install
install: install-foundry deps

.PHONY: install-rust
install-rust:
	curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

.PHONY: install-foundry
install-foundry:
	curl -L https://foundry.paradigm.xyz | bash
	~/.foundry/bin/foundryup --commit $(FOUNDRY_COMMIT)

.PHONY: deps
deps:  clean-lib forge-deps


##
# Solidity Setup
##
.PHONY: clean-lib
clean-lib:
	rm -rf lib

.PHONY: forge-deps
forge-deps:
	forge install --no-git \
		github.com/foundry-rs/forge-std \
	 	base-contracts=https://github.com/felipe-op/base-contracts@581773d3013498ceffe039f3384a8a4f4a3ef5d2 \
	 	https://github.com/ethereum-optimism/optimism@57413031bd75f497a5d64f357453d44804a1a77f

##
# Solidity Testing
##
.PHONY: solidity-test
solidity-test:
	forge test --ffi -vvv