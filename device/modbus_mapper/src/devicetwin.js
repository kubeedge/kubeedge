const constant = require('./constant');
const common = require('./common');
const Buffer = require('buffer').Buffer;
const uuidv4 = require('uuid/v4');
const util = require('util');
const async = require('async');

class DeviceTwin {
    constructor(mqttClient) {
        this.mqttClient = mqttClient;
    }
    
    // transferType transfer data according to the dpl configuration
    transferType(visitor, property, data, callback) {
        let transData;
        async.waterfall([
            function(callback) {
                if (visitor.visitorConfig.isRegisterSwap) {
                    common.switchRegister(data, (switchedData)=>{
                        callback(null, switchedData);
                    });
                } else {
                    callback(null, data);
                }
            },
            function(internalData, callback) {
                if (visitor.visitorConfig.isSwap && (visitor.visitorConfig.register === 'HoldingRegister' || visitor.visitorConfig.register === 'InputRegister')) {
                    common.switchByte(internalData, (switchedData)=>{
                        callback(null, switchedData);
                    });
                } else {
                    callback(null, internalData);
                }
            }
        ], function(err, transedData) {
            transData = transedData;
        });
        this.transferDataType(visitor, property, transData, callback);
    }

    // transferDataType transfer data according to the dpl configuration
    transferDataType(visitor, property, data, callback) {
        let transData;
        switch(property.dataType) {
            case 'int':
            case 'float':
                if (visitor.visitorConfig.register === 'DiscreteInputRegister' || visitor.visitorConfig.register === 'CoilRegister') {
                    common.bitArrayToInt(data, (num)=>{
                        transData = num;
                    });
                } else if (visitor.visitorConfig.register === 'HoldingRegister' || visitor.visitorConfig.register === 'InputRegister') {
                    common.twoByteArrayToInt(data, (num)=>{
                        transData = num;
                    })
                }

                if (visitor.visitorConfig.scale !=0 && transData != null) {
                    transData = transData * visitor.visitorConfig.scale;
                }

                if (property.dataType === 'int') {
                    transData = parseInt(transData);
                }

                if (property.maximum !== null && transData > parseFloat(property.maximum)) {
                    logger.info("read data is larger than max value, use max value")
                    transData = parseInt(property.maximum);
                } else if (property.minimum !== null && transData < parseFloat(property.minimum)) {
                    logger.info("read data is smaller than min value, use min value")
                    transData = parseInt(property.minimum);
                }

                callback(transData);
                break;
            case 'string':
                let buf = new Buffer.from(data);
                transData = buf.toString('utf8')
                callback(transData);
                break;
            case 'boolean':
                if (data[0] == 0 || data[0] == 1){
                    transData = Boolean(data[0]);
                } else {
                    transData = null;
                }
                callback(transData);
                break;
            default:
                logger.error('unknown dataType: ', property.dataType);
                callback(null);
                break;    
        }
    }

    // updateActual update actual value to edge mqtt
    updateActual(deviceID, property, value) {
        let reply_msg = {
            event_id: "",
            timestamp: new Date().getTime()
        };
        let twin = {};
        twin[property.name] = {
            actual: {
                value: String(value),
                metadata: {
                    timestamp: new Date().getTime()
                }
            },
            metadata: {
                tyep: property.dataType
            }
        };
        reply_msg.twin = twin;
        this.mqttClient.publish(constant.defaultTopicPrefix + deviceID + constant.twinUpdateTopic, JSON.stringify(reply_msg));
    }

    // dealUpdate set latest actual value of devicetwin into actualVal map
    dealUpdate(transData, property, deviceID, actualVals) {
        if (!actualVals.has(util.format("%s-%s", deviceID, property.name))) {
            this.updateActual(deviceID, property, transData);
            actualVals.set(util.format("%s-%s", deviceID, property.name), String(transData));
            logger.info("update devicetwin[%s] of device[%s] successfully", property.name, deviceID);
        } else {
            this.compareActuals(transData, actualVals.get(util.format("%s-%s", deviceID, property.name)), (changed)=>{
                if (changed) {
                    this.updateActual(deviceID, property, transData);
                    actualVals.set(util.format("%s-%s", deviceID, property.name), String(transData));
                    logger.info("update devicetwin[%s] of device[%s] successfully", property.name, deviceID);
                }
            });
        }
    }

    // getActuals publish get devicetwin msg to edge mqtt
    getActuals(deviceID) {
        let payload_msg = {
            event_id: "",
            timestamp: new Date().getTime()
        };
        this.mqttClient.publish(constant.defaultTopicPrefix + deviceID + constant.twinGetTopic, JSON.stringify(payload_msg));
    }

    // setActuals set device property and actual value map
    setActuals(getMsg, callback) {
        let deviceTwin = getMsg.twin;
        let PropActuals = new Map();
        Object.keys(deviceTwin).forEach(function(key){
            if (deviceTwin[key].hasOwnProperty('actual')) {
                PropActuals.set(key, deviceTwin[key].actual.value);
            }
        })
        callback(PropActuals);
    }

    // setExpecteds set device property and expected value map
    setExpecteds(getMsg, callback) {
        let deviceTwin = getMsg.twin;
        let ProExpect = new Map();
        Object.keys(deviceTwin).forEach(function(key){
            if (deviceTwin[key].hasOwnProperty('expected') && !deviceTwin[key].hasOwnProperty('actual') || JSON.stringify(deviceTwin[key].actual) == '{}') {
                ProExpect.set(key, deviceTwin[key].expected.value);
            }
        })
        callback(ProExpect);
    }

    // compareActuals compare if data is changed
    compareActuals(data, cachedActuals, callback) {
        let changed = false;
        if (data != cachedActuals) {
            changed = true;
        }
        callback(changed);
    }

    // UpdateDirectActuals update all devicetwin property to edge mqtt
    UpdateDirectActuals(devIns, deviceID, actualVals) {
        if (devIns.has(deviceID)) {
            let deviceName = devIns.get(deviceID).name;
            this.generateDirectGetMsg(deviceName, deviceID, actualVals, (directGetMsg)=>{
                this.mqttClient.publish(constant.defaultDirectTopicPrefix + deviceID + constant.directGetTopic, JSON.stringify(directGetMsg));
            });
        }
    }

    // generateDirectGetMsg generate Direct Get Msg in message format
    generateDirectGetMsg(deviceName, deviceID, actualVals, callback) {
        let header = {
            msg_id: uuidv4(),
            parent_msg_id: "",
            timestamp: new Date().getTime(),
            sync: false
        };
        let route = {
            source: "eventbus",
            group: "",
            operation: "upload",
            resource: util.format("%s%s%s", constant.defaultDirectTopicPrefix, deviceID, constant.directGetTopic)
        };
        let content = {
            data: actualVals,
            device_name: deviceName,
            device_id: deviceID,
            timestamp: new Date().getTime()
        };
        let directGetMsg = {
            header: header,
            route: route,
            content: content
        };
        callback(directGetMsg);
    }

    // syncExpected check whether expected value should be update to device
    static syncExpected(delta, key, callback) {
        let deviceTwin = delta.twin[key];
        if (!delta.twin.hasOwnProperty(key)) {
            logger.error("Invalid device twin ", key);
            return;
        }
        if (!deviceTwin.hasOwnProperty('actual') ||
          (deviceTwin.hasOwnProperty('expected') && deviceTwin.expected.hasOwnProperty('metadata') && deviceTwin.actual.hasOwnProperty('metadata') && 
            deviceTwin.expected.metadata.timestamp > deviceTwin.actual.metadata.timestamp &&
            deviceTwin.expected.value !== deviceTwin.actual.value)) {
          callback(deviceTwin.expected.value);
        }
    }
}

module.exports = DeviceTwin;
