// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;

import "./Pauseable.sol";
import "forge-std/console.sol";
import "@base-contracts/script/universal/MultisigBuilder.sol";
import {IGnosisSafe} from "@eth-optimism-bedrock/scripts/interfaces/IGnosisSafe.sol";

contract CallUnpause is MultisigBuilder {
    function _postCheck() internal view override {
        IGnosisSafe safe = IGnosisSafe(_ownerSafe());
        console.log("Nonce post check", safe.nonce());
    }

    function _buildCalls() internal view override returns (IMulticall3.Call3[] memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](1);

        calls[0] = IMulticall3.Call3({
            target: _superchainConfigAddr(),
            allowFailure: false,
            callData: abi.encodeCall(Pausable.unpause, ())
        });

        return calls;
    }

    function _ownerSafe() internal view override returns (address) {
        return vm.envAddress("SAFE_ADDR");
    }

    function _superchainConfigAddr() internal view returns (address) {
        return vm.envAddress("TARGET_ADDR");
    }
}
