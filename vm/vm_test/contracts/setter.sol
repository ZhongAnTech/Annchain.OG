pragma solidity ^0.4.20;

contract setter {
    uint public i;
    
    function setter() public {
        i = 10;
    }
    
    function helloWorld() public pure returns (string){
        return 'helloWorld';
    }
    
    function set(uint n) public {
        i = n;
    }
}