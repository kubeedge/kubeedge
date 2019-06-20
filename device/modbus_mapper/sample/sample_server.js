// create an empty modbus client
const ModbusRTU = require("modbus-serial");
var holdingValue = 5
var coilValue = false
var vector = {
    getHoldingRegister: function() {
        return holdingValue;
    },
    getCoil: function() {
        return coilValue;
    },
    setRegister: function(addr, value, unitID) {
        // Asynchronous handling supported also here
        console.log("set register", addr, value, unitID);
        holdingValue = value;
        return;
    },
    setCoil: function(addr, value, unitID) {
        // Asynchronous handling supported also here
        console.log("set coil", addr, value, unitID);
        coilValue = Boolean(value);
        return;
    },
};

// set the server to answer for modbus requests
console.log("ModbusTCP listening on modbus://127.0.0.1:5028");
var serverTCP = new ModbusRTU.ServerTCP(vector, { host: "0.0.0.0", port: 5028, debug: true, unitID: 1 });

serverTCP.on("socketError", function(err){
    // Handle socket error if needed, can be ignored
    console.error(err);
});