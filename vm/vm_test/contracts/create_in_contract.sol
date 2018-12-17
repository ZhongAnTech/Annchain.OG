pragma solidity 0.4.20;

/*
* @title Example the for Solidity Course
* @author Ethereum Community
*/

// A contract can create a new contract using the new keyword.
// The full code of the contract being created has to be known in advance,
// so recursive creation-dependencies are not possible.

// lets define a simple contract D with payable modifier
contract D {
    uint x;

    function D(uint a) public payable {
        x = a;
    }

    function getD() public view returns (uint){
        return x;
    }
}


contract C {
    // this is how you can create a contract using new keyword
    D d = new D(4);
    // this above line will be executed as part of C's constructor

    // you can also create a contract which is already defined inside a function like this
    function createD(uint arg) public {
        new D(arg);
    }


    // You can also create a contract and at the same time transfer some ethers to it.
    function createAndEndowD(uint arg, uint amount) public payable returns (uint){
        // Send ether along with the creation of the contract
        D newD = (new D).value(amount)(arg);
        return newD.getD();
    }
}