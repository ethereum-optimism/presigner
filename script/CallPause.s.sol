// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;

import "forge-std/console.sol";
import "@base-contracts/script/universal/MultisigBuilder.sol";
import { IGnosisSafe } from "@eth-optimism-bedrock/scripts/interfaces/IGnosisSafe.sol";

interface Pausable {
     function pause() external;
}

contract CallPause is MultisigBuilder {
 
    function _postCheck() internal override view {
        IGnosisSafe safe = IGnosisSafe(_ownerSafe());
        console.log("Nonce post check", safe.nonce());
    }

    function _buildCalls() internal override view returns (IMulticall3.Call3[] memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](1);

        calls[0] = IMulticall3.Call3({
            target: _optimismPortalAddr(),
            allowFailure: false,
            callData: abi.encodeCall(
                Pausable.pause,
                ()
            )
        });

        return calls;
    }

    function _ownerSafe() internal override view returns (address) {
        return vm.envAddress("SAFE_ADDR");
    }

    function _optimismPortalAddr() internal view returns (address) {
        return vm.envAddress("TARGET_ADDR");
    }

}