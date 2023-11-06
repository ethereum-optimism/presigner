ifndef $(GOPATH)
    GOPATH=$(shell go env GOPATH)
    export GOPATH
endif

.PHONY: install-foundry
install-foundry:
	curl -L https://foundry.paradigm.xyz | bash
	~/.foundry/bin/foundryup --commit $(FOUNDRY_COMMIT)

##
# Solidity Setup
##
.PHONY: deps
deps: install-eip712sign clean-lib forge-deps

.PHONY: install-eip712sign
install-eip712sign:
	go install github.com/base-org/eip712sign@v0.0.4

.PHONY: clean-lib
clean-lib:
	rm -rf lib

.PHONY: forge-deps
forge-deps:
	forge install --no-git \
		github.com/foundry-rs/forge-std@.. \
	 	base-contracts=https://github.com/base-org/contracts@4fee7d0a08b81e4041dd107140d63a55e4e79394 \
	 	https://github.com/ethereum-optimism/optimism@57413031bd75f497a5d64f357453d44804a1a77f

##
# Solidity Testing
##
.PHONY: solidity-test
solidity-test:
	forge test --ffi -vvv