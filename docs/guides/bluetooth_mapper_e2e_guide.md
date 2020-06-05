# Bluetooth Mapper End to End Test Setup Guide

The test setup required for running the end to end test of bluetooth mapper requires two separate machines in bluetooth range.
The paypal/gatt package used for bluetooth mapper makes use of hci interface for bluetooth communication. Out of two machines specified,
one is used for running bluetooth mapper and other is used for running a test server which publishes data that the mapper use for processing.
The test server created here is also using the paypal/gatt package.

## Steps for running E2E tests

1. Turn ON bluetooth service of both machines
2. Run server on first machine. Follow steps given below for running the test server.
3. For running mapper tests on second machine, clone kubbedge code and follow steps 4,5 and 6.
4. Update "dockerhubusername" and "dockerhubpassword" in tests/e2e/scripts/fast_test.sh with your credentials.
5. Compile bluetooth mapper e2e by executing the following command in $GOPATH/src/github.com/kubeedge/kubeedge.
`bash -x tests/e2e/scripts/compile.sh bluetooth`
6. Run bluetooth mapper e2e by executing the following command in $GOPATH/src/github.com/kubeedge/kubeedge.
`bash -x tests/e2e/scripts/execute.sh bluetooth`

#### Test Server Creation

1. Copy devices folder in tests/e2e/stubs and keep it in path TESTSERVER/src/github.com in first machine.
2. Update the following in devices/mockserver.go

    1. package devices -> package main
    2. import "github.com/kubeedge/kubeedge/tests/stubs/devices/services" to "github.com/devices/services"

3. Build the binary using
`go build mockserver.go`
4. Run the server using
`sudo ./mockserver -logtostderr -duration=<specify duration for which test server should be running>`

 _sudo is required for getting hci control of the machine._

This runs your test server which publishes data for the mapper to process.




