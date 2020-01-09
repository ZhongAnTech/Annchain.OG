pragma solidity ^0.4.0;

contract Caller {
    uint public value;
    address public msgSender;
    address public txOrigin;

    function callSetValue(address _callee, uint _value) {
        _callee.call(bytes4(keccak256("setValue(uint256)")), _value); // Callee's storage is set as given , Caller's is not modified
    }

    function callcodeSetValue(address _callee, uint _value) {
        _callee.callcode(bytes4(keccak256("setValue(uint256)")), _value); // Caller's storage is set, Calee is not modified
    }

    function delegatecallSetValue(address _callee, uint _value) {
        _callee.delegatecall(bytes4(keccak256("setValue(uint256)")), _value); // Caller's storage is set, Callee is not modified
    }
}

contract Callee {
    uint public value;
    address public msgSender;
    address public txOrigin;


    function setValue(uint _value) {
        value = _value;
        msgSender = msg.sender;
        txOrigin = tx.origin;
        // msg.sender is Caller if invoked by Caller's callcodeSetValue. None of Callee's storage is updated
        // msg.sender is OnlyCaller if invoked by onlyCaller.justCall(). None of Callee's storage is updated

        // the value of "this" is Caller, when invoked by either Caller's callcodeSetValue or CallHelper.justCall()
    }
}

contract CallHelper {
    function justCall(Caller _caller, Callee _callee, uint _value) {
        _caller.delegatecallSetValue(_callee, _value);
    }
}