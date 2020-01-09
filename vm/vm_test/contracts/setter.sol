pragma solidity ^0.4.20;

contract setter {
    uint public i;
    uint public blc;
    
    function setter() public {
        i = 10;
        blc = 0;
    }
    
    function() public payable {
        blc = msg.value;
    }
    
    function helloWorld() public pure returns (string){
        return 'helloWorld';
    }
    
    function set(uint n) public {
        i = n;
    }
}