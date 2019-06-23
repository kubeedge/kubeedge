const async = require('async');
const log4js = require('log4js');
const mqtt = require('mqtt');
const path = require('path');
const util = require('util');

const common = require('./common');
const constant = require('./constant');
const DeviceTwin = require('./devicetwin');
const Modbus = require('./modbus');
const WatchFiles = require('./watchfile');

//default logger options
log4js.configure({
    appenders: {
        out: { type: 'stdout' },
    },
    categories: {
        default: { appenders: ['out'], level: 'info' }
    }
});
logger = log4js.getLogger('appenders');

let options = {
    port: 1883,
    host: '127.0.0.1',
    dpl_name: 'dpl/deviceProfile.json'
};

let mqtt_client, mqtt_client2, msg, mqtt_options, dt, devIns, devMod, devPro, modVistr;
let ActualVal = new Map();

async.series([
    //load conf.json
    function(callback) {
        WatchFiles.loadConfig('conf/conf.json', (err, configs)=>{
            if (err) {
                logger.error('failed to load config, err: ', err);
            } else {
                options = {
                    port: configs.mqtt_port,
                    host: configs.mqtt_ip,
                    dpl_name: configs.dpl_name
                };
                callback(err);
            }
        });
    },
    
    //load dpl first time
    function(callback) {
        WatchFiles.loadDpl(options.dpl_name, (devInsMap, devModMap, devProMap, modVistrMap)=>{
            devIns = devInsMap;
            devMod = devModMap;
            devPro = devProMap;
            modVistr = modVistrMap;
            callback();
        });
    },

    //first get twinget build map
    function(callback) {
        mqtt_options = {
            port: options.port,
            host: options.host,
        };
        mqtt_client = mqtt.connect(mqtt_options);
        dt = new DeviceTwin(mqtt_client);
        mqtt_client.on('connect', ()=>{
            logger.info('connetced to edge mqtt with topic twinGet');
            mqtt_client.subscribe(constant.twinGetResTopic);
            for (let instance of devIns) {
                dt.getActuals(instance[0]);
            }
        });
        callback();
    },

    //deal with twin get msg and set expected value into device
    function(callback) {
        mqtt_client.on('message', (topic, message)=>{
            try {
                var msgGet = JSON.parse(message.toString());
            } catch (err) {
                logger.error('unmarshal error');
                return;
            }
            let resources = topic.toString().split('/');
            let deviceID = resources[3];
            let dt = new DeviceTwin(mqtt_client);
            let devProtocol, devInstance;
            if (devPro.has(deviceID) && devIns.has(deviceID)) {
                devProtocol = devPro.get(deviceID);
                devInstance = devIns.get(deviceID);
            } else {
                logger.error('match visitor failed');
            }
            logger.info('recieve twinGet msg, set properties actual value map');
            if (resources.length === 7 && resources[5] === 'get' && msgGet != null && msgGet.code != 404 && typeof(devProtocol) != 'undefined' && typeof(devInstance) != 'undefined') {
                dt.setActuals(msgGet, (PropActuals)=>{
                    for (let actual of PropActuals) {
                        ActualVal.set(util.format('%s-%s', deviceID, actual[0]), actual[1]);
                    }
                });
                dt.setExpecteds(msgGet, (PropExpecteds)=>{
                    for (let expected of PropExpecteds) {
                        modbusProtocolTransfer(devProtocol.protocol, (transferedProtocol)=>{
                            if (modVistr.has(util.format('%s-%s-%s', devInstance.model, expected[0], transferedProtocol))) {
                                let visitor = modVistr.get(util.format('%s-%s-%s', devInstance.model, expected[0], transferedProtocol));
                                dealDeltaMsg(msgGet, expected[0], visitor, devProtocol, expected[1]);
                            }
                        });
                    }
                });
            }
        });
        callback();
    },

    // start mqtt sub delta topic
    function(callback) {
        mqtt_options = {
            port: options.port,
            host: options.host,
        };
        mqtt_client2 = mqtt.connect(mqtt_options);
        mqtt_client2.on('connect', ()=>{
            logger.info('connetced to edge mqtt with topic twinDelta');
            mqtt_client2.subscribe(constant.twinDeltaTopic);
        });
        callback();
    },

    // on receive msg of delta topic
    function(callback) {
        logger.info('start to wait for devicetwin update');
        mqtt_client2.on('message', (topic, message)=>{
            try {
                msg = JSON.parse(message.toString());
            } catch (err) {
                logger.error('unmarshal error');
                callback(err);
                return;
            }

            //match visitors
            let resources = topic.toString().split('/');
            let deviceID = resources[3];
            let devProtocol, devInstance;
            if (devPro.has(deviceID) && devIns.has(deviceID)) {
                devProtocol = devPro.get(deviceID);
                devInstance = devIns.get(deviceID);
            } else {
                logger.error('match visitor failed');
            }

            try {
                if (resources.length === 7 && resources[6] === 'delta' && typeof(devProtocol) != 'undefined' && typeof(devInstance) != 'undefined') {
                    logger.info('recieved twinDelta msg');
                    Object.keys(msg.delta).forEach(function(key){
                        modbusProtocolTransfer(devProtocol.protocol, (transferedProtocol)=>{
                            if (modVistr.has(util.format('%s-%s-%s', devInstance.model, key, transferedProtocol))) {
                                let visitor = modVistr.get(util.format('%s-%s-%s', devInstance.model, key, transferedProtocol));
                                DeviceTwin.syncExpected(msg, key, (value)=>{
                                    dealDeltaMsg(msg, key, visitor, devProtocol, value);
                                });
                            }
                        });
                    });
                }  
            } catch (err) {
                logger.error('failed to change devicetwin of device[%s], err: ', deviceID, err);
            }
        });
    }
],function(err) {
    if (err) {
        logger.error(err);
    } else {
        logger.info('changed devicetwin successfully');
    }
});

let mqtt_client3;
logger.info('start to watch dpl config');
WatchFiles.watchChange(path.join(__dirname, 'dpl'), ()=>{
    async.series([
        function(callback) {
            WatchFiles.loadDpl(options.dpl_name, (devInsMap, devModMap, devProMap, modVistrMap)=>{
                devIns = devInsMap;
                devMod = devModMap;
                devPro = devProMap;
                modVistr = modVistrMap;
                callback();
            });
        },

        function(callback) {
            mqtt_options = {
                port: options.port,
                host: options.host,
            };
            mqtt_client3 = mqtt.connect(mqtt_options);
            let dt = new DeviceTwin(mqtt_client3);
            mqtt_client3.on('connect', ()=>{
                logger.info('connetced to edge mqtt with topic twinGet');
                mqtt_client.subscribe(constant.twinGetResTopic);
                for (let instance of devIns) { // let change from var
                    dt.getActuals(instance[0]);
                }
            });
            callback();
        },

        function(callback) {
            mqtt_client3.on('message', (topic, message)=>{
                try {
                    var msgGet = JSON.parse(message.toString());
                } catch (err) {
                    logger.error('unmarshal error');
                    return;
                }
                let resources = topic.toString().split('/');
                let deviceID = resources[3];
                let devProtocol, devInstance;
                if (devPro.has(deviceID) && devIns.has(deviceID)) {
                    devProtocol = devPro.get(deviceID);
                    devInstance = devIns.get(deviceID);
                } else {
                    logger.error('match visitor failed');
                }

                let dt = new DeviceTwin(mqtt_client3);
                if (resources.length === 7 && resources[5] === 'get' && msgGet != null && msgGet.code != 404 && typeof(devProtocol) != 'undefined' && typeof(devInstance) != 'undefined') {
                    logger.info('received twinGet message');
                    dt.setExpecteds(msgGet, (PropExpecteds)=>{
                        for (let expected of PropExpecteds) {
                            dt.compareActuals(expected[1], ActualVal.get(util.format('%s-%s', deviceID, expected[0])),(changed)=>{
                                modbusProtocolTransfer(devProtocol.protocol, (transferedProtocol)=>{
                                    if (changed && modVistr.has(util.format('%s-%s-%s', devInstance.model, expected[0], transferedProtocol))) {
                                        let visitor = modVistr.get(util.format('%s-%s-%s', devInstance.model, expected[0], transferedProtocol));
                                        dealDeltaMsg(msgGet, expected[0], visitor, devProtocol, expected[1]);
                                    }
                                });
                            });
                        }
                    });
                }
            });
            callback();    
        }
    ],function(err) {
        if (err) {
            logger.error('failed to load changed dpl config, err: ', err);
        } else {
            logger.info('load changed dpl config successfully');
        }
    });
});

logger.info('start to check devicetwin state');
setInterval(()=>{
    let dt = new DeviceTwin(mqtt_client);
    logger.info('chechking devicetwin state');
    for (let instance of devIns) {
        if (devPro.has(instance[0])) {
            let protocol = devPro.get(instance[0]);
            let actuals = new Map();
            syncDeviceTwin(dt, instance[0], protocol, actuals);
        }
    }
}, 2000);

// syncDeviceTwin check each property of each device accroding to the dpl configuration
function syncDeviceTwin(dt, key, protocol, actuals) {
    async.eachSeries(devMod.get(key).properties, (property, callback)=>{
        let visitor;
        if (typeof(protocol) != 'undefined') {
            modbusProtocolTransfer(protocol.protocol, (transferedProtocol)=>{
                if (devIns.has(key) && modVistr.has(util.format('%s-%s-%s', devIns.get(key).model, property.name, transferedProtocol))) {
                    visitor = modVistr.get(util.format('%s-%s-%s', devIns.get(key).model, property.name, transferedProtocol));
                } else {
                    logger.error('failed to match visitor');
                }
            });
        }
        if (typeof(protocol) != 'undefined' && typeof(visitor) != 'undefined') {
            let modbus = new Modbus(protocol, visitor);
            modbus.ModbusUpdate((err, data)=>{
                if (err) {
                    logger.error('failed to update devicetwin[%s] of device[%s], err: ', property.name, key, err);
                } else {
                    dt.transferType(visitor, property, data, (transData)=>{
                        if (transData != null) {
                            actuals.set(property.name, String(transData));
                            dt.dealUpdate(transData, property, key, ActualVal);
                        }
                    });
                    callback();
                }
            });
        }
    },()=>{
        dt.UpdateDirectActuals(devIns, key, actuals);
    });
}

// dealDeltaMsg deal with the devicetwin delta msg
function dealDeltaMsg(msg, key, visitor, protocol, value) {
    let modbus = new Modbus(protocol, visitor);
    modbus.ModbusDelta(msg.twin[key].metadata.type, value, (err, data)=>{
        if (err) {
            logger.error('failed to modify register, err: ', err)
        } else {
            logger.info('modify register %s successfully', JSON.stringify(data));
        }
    })
}

function modbusProtocolTransfer(protocol, callback) {
    let transferedProtocol;
    if (protocol === 'modbus-rtu' || protocol === 'modbus-tcp') {
        transferedProtocol = 'modbus';
    } else {
        transferedProtocol = protocol;
    }
    callback(transferedProtocol)
}
