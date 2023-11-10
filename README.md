# Presigner

Multisig transaction presigner made easy <3

## Setup

1. Install [golang](https://golang.org/doc/install)

1. Install Foundry
```bash
make install-foundry
```

3. Download Foundry dependencies
```bash
make deps
```

## Usage

This tool is used to create and sign transactions for multisig safe calls.

The transactions are created and verified using [solidity scripting](https://book.getfoundry.sh/tutorials/solidity-scripting) by calling
the actual [safe contract](https://github.com/safe-global/safe-contracts/tree/main/contracts) functions.
It stores state in a single self-contained JSON file.

Once the transaction is fully signed, the `simulate` command produces a oneliner
shell script encoded in Base64 that can be easily stored in secret vaults for later use.
The onliner has dependency only on `cast` ([from Foundry](https://book.getfoundry.sh/reference/cast/cast-send)).

### File format

```json
{
    "chain_id": "5",
    "created_at": "2023-11-06T14:53:30-08:00",
    "data": "0x1901c0d0e680d49115459ede72891964cf5adc2cf1930f57e7d8f7cf2408ed63d6ad81b0007322861e475d3f147da54ca8278d8f2850deaf5c736817f679a65332fc",
    "rpc_url": "https://ethereum-goerli.publicnode.com",
    "safe_addr": "0xb7b28ac0c0ffab4188826b14d02b17e8b444ed9e",
    "safe_nonce": "3",
    "script_name": "CallPause",
    "signatures": [
        {
            "signer": "0x1234567890123456789012345678901234567890",
            "signature": "1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"
        },
        {
            "signer": "0x1234567890123456789012345678901234567891",
            "signature": "2111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"
        }
    ],
    "target_addr": "0x95B78e7A9f856161B8fE255Cf92C38d693aC6f5e"
}
```

### Commands

### nonce

Verifies the current nonce of a safe, example:

```bash
go run presigner.go \
    --safe-addr 0xb7b28ac0c0ffab4188826b14d02b17e8b444ed9e \
    nonce
```

#### create

Creates a new transaction to be signed, example:

```bash
go run presigner.go \
    --json-file tx/2023-11-06-goerli-pause-3.json \
    --chain 5 \
    --rpc-url https://ethereum-goerli.publicnode.com \
    --target-addr 0xfAF96f23026CA4863B6dcA30204aD5D2675738b8 \
    --safe-addr 0xb7b28ac0c0ffab4188826b14d02b17e8b444ed9e \
    --safe-nonce 3 \
    create

2023/11/06 13:12:32 saved: tx/2023-11-06-goerli-pause-3.json
```

Customizing the `safe-nonce` parameter it is possible to create transactions in advance.

### sign

Signs a transaction previously created, example:

```bash
go run presigner.go \
    --json-file tx/2023-11-06-goerli-pause-3.json \
    --private-key 0000000000000000000000000000000000000000000000000000000000000000 \
    sign

2023/11/06 13:12:42 added signature for 0x1234567890123456789012345678901234567890
```

As new signatures are added, the transaction is updated and saved.

We use [eip712signer](https://github.com/base-org/eip712signer) to sign the transaction, which currently supports:
* private-key
* ledger
* mnemonic

### verify

Verifies if a transaction previously created has valid signatures to be executed, example:

```bash
go run presigner.go \
    --json-file tx/2023-11-06-goerli-pause-3.json \
    verify
```

### simulate

Simulate the transaction execution in a forked VM, example:

```bash
go run presigner.go \
    --json-file tx/2023-11-06-goerli-pause-3.json \
    simulate
```

If the simulation succeeds, it will also print a cheat sheet to execute the transaction in the network, i.e.:

```bash
- - 8< - -

EXECUTORKEY=********
go run presigner.go \
    --json-file tx/2023-11-06-goerli-pause-3.json \
    --private-key $EXECUTORKEY \
    execute

- - 8< - -
```

Or you can use the oneliner script encoded in base64, i.e.:

```bash
- - 8< - -
/bin/bash <(base64 -d -i tx/2023-11-06-goerli-pause-3.sh.b64) --rpc-url https://ethereum-goerli.publicnode.com
- - 8< - -
```

The onliner script is a single line shell script that uses `cast` to execute the transaction.

The arguments passed to the oneliner script are passed to `cast send`,
so you can provide keys with `--ledger`, `--private-key` or `--menmonics`,
override the `--rpc-url`
and customize gas parameters at time of execution.

To see all options, just run `cast send --help`.

To double-check the contents of the oneliner script, you can use:
```bash
base64 -d -i tx/2023-11-07-goerli-pause-3.sh.b64
```

### execute

Execute the transaction in the network, example:

```bash
go run presigner.go \
    --json-file tx/2023-11-06-goerli-pause-3.json \
    --private-key 0000000000000000000000000000000000000000000000000000000000000000 \
    execute
```

Note you need a private-key to execute the transaction, but it does not need to be a signer.


## Safe error codes

From [safe-contracts](https://github.com/safe-global/safe-contracts/blob/main/docs/error_codes.md) repo:

### General init related
- `GS000`: `Could not finish initialization`
- `GS001`: `Threshold needs to be defined`
- `GS002`: `A call to set up modules couldn't be executed because the destination account was not a contract`

### General gas/ execution related
- `GS010`: `Not enough gas to execute Safe transaction`
- `GS011`: `Could not pay gas costs with ether`
- `GS012`: `Could not pay gas costs with token`
- `GS013`: `Safe transaction failed when gasPrice and safeTxGas were 0`

### General signature validation related
- `GS020`: `Signatures data too short`
- `GS021`: `Invalid contract signature location: inside static part`
- `GS022`: `Invalid contract signature location: length not present`
- `GS023`: `Invalid contract signature location: data not complete`
- `GS024`: `Invalid contract signature provided`
- `GS025`: `Hash has not been approved`
- `GS026`: `Invalid owner provided`

### General auth related
- `GS030`: `Only owners can approve a hash`
- `GS031`: `Method can only be called from this contract`

### Module management related
- `GS100`: `Modules have already been initialized`
- `GS101`: `Invalid module address provided`
- `GS102`: `Module has already been added`
- `GS103`: `Invalid prevModule, module pair provided`
- `GS104`: `Method can only be called from an enabled module`
- `GS105`: `Invalid starting point for fetching paginated modules`
- `GS106`: `Invalid page size for fetching paginated modules`

### Owner management related
- `GS200`: `Owners have already been set up`
- `GS201`: `Threshold cannot exceed owner count`
- `GS202`: `Threshold needs to be greater than 0`
- `GS203`: `Invalid owner address provided`
- `GS204`: `Address is already an owner`
- `GS205`: `Invalid prevOwner, owner pair provided`

### Guard management related
- `GS300`: `Guard does not implement IERC165`

### Fallback handler related
- `GS400`: `Fallback handler cannot be set to self`