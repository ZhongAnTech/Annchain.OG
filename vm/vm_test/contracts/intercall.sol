contract C1 {
    function f1() pure public returns(uint) {
        return(10);
    }
}

contract C2 {
    function f2(address addrC1) pure public returns(uint) {
        C1 c1 = C1(addrC1);
        return c1.f1();
    }
}