// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;

import "./Pauseable.sol";
import "forge-std/console.sol";
import "@base-contracts/script/universal/MultisigBuilder.sol";
import {IGnosisSafe} from "@eth-optimism-bedrock/scripts/interfaces/IGnosisSafe.sol";

contract LowLevelCallScript is MultisigBuilder {
    function _postCheck() internal view override {
        IGnosisSafe safe = IGnosisSafe(_ownerSafe());
        console.log("Nonce post check", safe.nonce());
    }

    function _buildCalls() internal view override returns (IMulticall3.Call3[] memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](1);

        bytes memory toCall = vm.envBytes("TX_CALLDATA");

        calls[0] = IMulticall3.Call3({
            target: _targetAddress(),
            allowFailure: false,
            callData: toCall 
        });

        return calls;
    }

    function _ownerSafe() internal view override returns (address) {
        return vm.envAddress("SAFE_ADDR");
    }

    function _targetAddress() internal view returns (address) {
        return vm.envAddress("TARGET_ADDR");
    }
}
