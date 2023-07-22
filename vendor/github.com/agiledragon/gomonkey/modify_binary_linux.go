package gomonkey

import "syscall"

func modifyBinary(target uintptr, bytes []byte) {
    function := entryAddress(target, len(bytes))
    
    page := entryAddress(pageStart(target), syscall.Getpagesize())
    err := syscall.Mprotect(page, syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC)
    if err != nil {
        panic(err)
    }
    copy(function, bytes)
    
    err = syscall.Mprotect(page, syscall.PROT_READ|syscall.PROT_EXEC)
    if err != nil {
        panic(err)
    }
}
