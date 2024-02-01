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
		github.com/OpenZeppelin/openzeppelin-contracts@v4.9.3 \
		github.com/OpenZeppelin/openzeppelin-contracts-upgradeable@v4.7.3 \
		github.com/rari-capital/solmate@8f9b23f8838670afda0fd8983f2c41e8037ae6bc \
		github.com/Vectorized/solady@862a0afd3e66917f50e987e91886b9b90c4018a1 \
	 	base-contracts=https://github.com/base-org/contracts@c2a45d575fbf2a7c41370aaa1e9d47a3f6020576 \
	 	https://github.com/ethereum-optimism/optimism@57413031bd75f497a5d64f357453d44804a1a77f

##
# Solidity Testing
##
.PHONY: solidity-test
solidity-test:
	forge test --ffi -vvv