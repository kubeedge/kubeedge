# Unit Test Guide
The purpose of this document is to give introduction about unit tests and to help contributors in writing unit tests.

## Unit Test  
 
Read this [article](http://softwaretestingfundamentals.com/unit-testing/) for a simple introduction about unit tests and benefits of unit testing. Go has its own built-in package called testing and command called ```go test```.  
For more detailed information on golang's builtin testing package read this [document](https://golang.org/pkg/testing/]).
 
## Mocks  

 The object which needs to be tested may have dependencies on other objects. To confine the behavior of the object under test, replacement of the other objects by mocks that simulate the behavior of the real objects is necessary.
 Read this [article](https://medium.com/@piraveenaparalogarajah/what-is-mocking-in-testing-d4b0f2dbe20a) for more information on mocks.
 
 GoMock is a mocking framework for Go programming language.
 Read [godoc](https://godoc.org/github.com/golang/mock/gomock) for more information about gomock.
 
 Mock for an interface can be automatically generated using [GoMocks](https://github.com/golang/mock) mockgen package.
 
 **Note** There is gomock package in kubeedge vendor directory without mockgen. Please use mockgen package of tagged version ***v1.1.1*** of [GoMocks github repository](https://github.com/golang/mock) to install mockgen and generate mocks. Using higher version may cause errors/panics during execution of you tests.

There is gomock package in kubeedge vendor directory without mockgen. Please use mockgen package of tagged version ***v1.1.1*** of [GoMocks github repository](https://github.com/golang/mock) to install mockgen and generate mocks. Using higher version may cause errors/panics during execution of you tests.

 Read this [article](https://blog.codecentric.de/en/2017/08/gomock-tutorial/) for a short tutorial of usage of gomock and mockgen.
 
## Ginkgo  
  
 [Ginkgo](https://onsi.github.io/ginkgo/) is one of the most popular framework for writing tests in go.
 
 Read [godoc](https://godoc.org/github.com/onsi/ginkgo) for more information about ginkgo.
 
See a [sample](https://github.com/kubeedge/kubeedge/blob/master/edge/pkg/metamanager/dao/meta_test.go) in kubeedge where go builtin package testing and gomock is used for writing unit tests.

See a [sample](https://github.com/kubeedge/kubeedge/blob/master/edge/pkg/devicetwin/dtmodule/dtmodule_test.go) in kubeedge where ginkgo is used for testing.

## Writing UT using GoMock  

### Example : metamanager/dao/meta.go  

After reading the code of meta.go, we can find that there are 3 interfaces of beego which are used. They are [Ormer](https://github.com/kubeedge/kubeedge/blob/master/vendor/github.com/astaxie/beego/orm/types.go), [QuerySeter](https://github.com/kubeedge/kubeedge/blob/master/vendor/github.com/astaxie/beego/orm/types.go) and [RawSeter](https://github.com/kubeedge/kubeedge/blob/master/vendor/github.com/astaxie/beego/orm/types.go).

We need to create fake implementations of these interfaces so that we do not rely on the original implementation of this interface and their function calls.

Following are the steps for creating fake/mock implementation of Ormer, initializing it and replacing the original with fake.  

1. Create directory mocks/beego.  

2. use mockgen to generate fake implementation of the Ormer interface
```shell
mockgen -destination=mocks/beego/fake_ormer.go -package=beego github.com/astaxie/beego/orm Ormer
```
- `destination` : where you want to create the fake implementation.  
- `package` : package of the created fake implementation file  
- `github.com/astaxie/beego/orm` : the package where interface definition is there  
- `Ormer` : generate mocks for this interface

3. Initialize mocks in your test file. eg meta_test.go
```shell
mockCtrl := gomock.NewController(t)
defer mockCtrl.Finish()
ormerMock = beego.NewMockOrmer(mockCtrl)
```  

4. ormermock is now a fake implementation of Ormer interface. We can make any function in ormermock return any value you want.    

5. replace the real Ormer implementation with this fake implementation. DBAccess is variable to type Ormer which we will replace with mock implemention  
```shell
dbm.DBAccess = ormerMock
```   

6. If we want Insert function of ormer interface which has return types as (int64,err) to return (1 nil), it can be done in 1 line in your test file using gomock.  
```shell
ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
```  

``Expect()`` : is to tell that a function of ormermock will be called.

``Insert(gomock.Any())`` : expect Insert to be called with any parameter.

``Return(int64(1), nil)`` : return 1 and error nil

``Times(1)``: expect insert to be called once and return 1 and nil only once.

So whenever insert is called, it will return 1 and nil, thus removing the dependency on external implementation.