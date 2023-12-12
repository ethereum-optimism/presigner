// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;

interface Pausable {
    function pause() external;
    function unpause() external;
}