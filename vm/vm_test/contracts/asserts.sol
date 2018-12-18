pragma solidity ^0.4.20;

contract asserts {

    mapping (address => address) public calledby;          // (addr => pID) record if called successfully

    function req() payable public {
        calledby[msg.sender] = msg.sender;
        // gas stop
        require(msg.value % 2 == 0);
    }
    function asrt() payable public {
        calledby[msg.sender] = msg.sender;
        // gas all gone
        assert(msg.value % 2 == 0);
    }
}
