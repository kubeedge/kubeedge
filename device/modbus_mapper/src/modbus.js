const ModbusRTU = require('modbus-serial');
const Buffer = require('buffer').Buffer;
const async = require('async');
const common = require('./common');

class Modbus {
    constructor(protocol, visitor){
        this.protocol = protocol;
        this.visitor = visitor;
        this.client = new ModbusRTU();
    }

    // connect to device with modbus protocol
    connect(callback) {
        let protocol = this.protocol;
        let client = this.client;
        switch(protocol.protocol) {
            case 'modbus-tcp':
                client.connectTCP(protocol.protocol_config.ip, { port: parseInt(protocol.protocol_config.port) }, ()=>{
                    client.setTimeout(500);
                    client.setID(parseInt(protocol.protocol_config.slaveID));
                    callback(client);
                });
                break;
            case 'modbus-rtu':
                async.series([
                    function(callback) {
                        setTimeout(function () {
                           callback(null, null);
                        }, 100);
                    },
                    function (callback) {
                        client.connectRTUBuffered(protocol.protocol_config.serialPort, { baudRate: parseInt(protocol.protocol_config.baudRate) }, ()=>{
                            client.setTimeout(500);
                            client.setID(parseInt(protocol.protocol_config.slaveID));
                            callback(null, client);
                        });
                    }], (err, res) => {callback(res[1]);
                });
                break;
            default:
                logger.info('unknwon modbus_type ', protocol.protocol);
                break;
        }
    }

    // WriteAction write the value into device registers
    WriteAction(value, callback) {
        let visitor = this.visitor;
        let client = this.client;
        switch(visitor.visitorConfig.register){
        case 'CoilRegister':
            client.writeCoils(parseInt(visitor.visitorConfig.index), value, (err, data)=>{
                client.close();
                callback(err, data);
            });
            break;
        case 'HoldingRegister':
            client.writeRegisters(parseInt(visitor.visitorConfig.index), value, (err, data)=>{
                client.close();
                callback(err, data);
            });
            break;
        default:
            client.close();
            logger.info('write action is not allowed on register type ', visitor.visitorConfig.register)
            callback('unkown action', null);
            break;
        }
    }

    // PreWriteAction transfer data before writing data to register
    PreWriteAction(type, value, callback) {
        let visitor = this.visitor;
        let transData;
        async.waterfall([
            function(callback) {
                switch(type) {
                    case 'int':
                    case 'float':
                        value = parseInt(value);
                        if (visitor.visitorConfig.register === 'CoilRegister') {
                            transData = (value).toString(2).split('').map(function(s) { return parseInt(s); });
                        } else if (visitor.visitorConfig.register === 'HoldingRegister') {
                            common.IntToByteArray(value, (byteArr)=>{
                                if (byteArr.length < visitor.visitorConfig.offset) {
                                    let zeroArr = new Array(visitor.visitorConfig.offset -byteArr.length).fill(0);
                                    byteArr = zeroArr.concat(byteArr);
                                    transData = byteArr;
                                } else {
                                    transData = byteArr;
                                }
                            });
                        } else {
                            transData = null;
                        }
                        callback(null, transData);
                        break;
                    case 'string': {
                        let buf = new Buffer.from(value);
                        transData = buf.toJSON().data;
                        callback(null, transData);
                        break;
                    }
                    case 'boolean':
                        if (value === 'true') {
                            transData = [1];
                        } else if (value === 'false') {
                            transData = [0];
                        } else {
                            transData = null;
                        }
                        callback(null, transData);
                        break;
                    default:
                        transData = null;
                        callback(null, transData);
                        break;
                }
            },
            function(transData, callback) {
                if (visitor.visitorConfig.isRegisterSwap && transData != null) {
                    common.switchRegister(transData, (switchedData)=>{
                        callback(null, switchedData);
                    });
                } else {
                    callback(null, transData);
                }
            },
            function(internalData, callback) {
                if (visitor.visitorConfig.isSwap && internalData != null && (visitor.visitorConfig.register === 'HoldingRegisters' || visitor.visitorConfig.register === 'CoilsRegisters')) {
                    common.switchByte(internalData, (switchedData)=>{
                        callback(null, switchedData);
                    });
                } else {
                    callback(null, internalData);
                }
            }], function(err, transData) {
                callback(transData);
            }
        );
    }

    // ReadAction read register data from device
    ReadAction(callback) {
        let visitor = this.visitor;
        let client = this.client;
        switch (visitor.visitorConfig.register) {
            case 'CoilRegister':
                client.readCoils(parseInt(visitor.visitorConfig.index), parseInt(visitor.visitorConfig.offset), (err, data)=>{
                    client.close();
                    callback(err, err?data:[data.data[0]]);
                });
                break;
            case 'DiscreteInputRegister':
                client.readDiscreteInputs(parseInt(visitor.visitorConfig.index), parseInt(visitor.visitorConfig.offset), (err, data)=>{
                    client.close();
                    callback(err, err?data:[data.data[0]]);
                });
                break;
            case 'HoldingRegister':
                client.readHoldingRegisters(parseInt(visitor.visitorConfig.index), parseInt(visitor.visitorConfig.offset), (err, data)=>{
                    client.close();
                    callback(err, err?data:data.data);
                });
                break;
            case 'InputRegister':
                client.readInputRegisters(parseInt(visitor.visitorConfig.index), parseInt(visitor.visitorConfig.offset), (err, data)=>{
                    client.close();
                    callback(err, err?data:data.data);
                });
                break;
            default:
                client.close();
                logger.info('read action is not allowed on register type ', visitor.visitorConfig.register)
                callback('unknown Registers type', null);
                break;
        }
    }

    // ModbusDelta deal with the delta message to modify the register
    ModbusDelta(type, value, callback) {
        this.connect(()=>{
            this.PreWriteAction(type, value, (transData)=>{
                if (transData != null) {
                    this.WriteAction(transData, (err, data)=>{
                        callback(err, data);
                    });
                }
            });
        });
    }

    // ModbusUpdate deal with the update message to read the register
    ModbusUpdate(callback) {
        this.connect(()=>{
            this.ReadAction((err, data)=>{
                callback(err, data);
            });
        });
    }
}

module.exports = Modbus;
