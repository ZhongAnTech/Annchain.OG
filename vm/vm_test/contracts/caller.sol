pragma solidity ^0.4.20;

interface setter {
    function set(uint n) external;
}

contract caller {
    
    address public callee;
    setter public ster;
    
    function caller() public {
        callee = 0x2fd82147682e011063adb8534b2d1d8831f52969;
        ster = setter(callee);
    }
    
    function callSetter(uint256 value) public {
        ster.set(value);
    }
}

