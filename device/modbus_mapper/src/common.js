const fs = require('fs');
const path = require('path');
const util = require('util');
const mkdirp = require('mkdirp');

// bitArrayToInt change bit array to Int
function bitArrayToInt(bitArr, callback) {
    let bitStr = '';
    if (bitArr.length > 0 && bitArr.length < 64) {
        for (let i = 0; i < bitArr.length; i++){
            bitStr = bitStr + bitArr[i].toString();
        }
        num = parseInt(bitStr,2);
        callback(num);
    }
}

// byteArrayToInt change one byte array to Int
function byteArrayToInt(byteArr, callback) {
    let bitArr = '';
    if (byteArr.length > 0 && byteArr.length < 5){
        for (let i = 0; i < byteArr.length; i++){
            bitArr = bitArr + (byteArr[i]).toString(2).padStart(8, '0');
        }
        callback(parseInt(bitArr, 2));
    }
}

// twoByteArrayToInt change two byte array to Int
function twoByteArrayToInt(byteArr, callback) {
    let bitArr = '';
    if (byteArr.length > 0 && byteArr.length < 5){
        for (let i = 0; i < byteArr.length; i++){
            bitArr = bitArr + (byteArr[i]).toString(2).padStart(16, '0');
        }
        callback(parseInt(bitArr, 2));
    }
}

// IntToByteArray change Int to byte array
function IntToByteArray(value, callback) {
    if ((value).toString(2).length > 32){
        let cs1 = (value).toString(2).slice(0,(value).toString(2).length-32);
        let cs2 = (value).toString(2).slice((value).toString(2).length-32);
        Int32ToByte(parseInt(cs1, 2), (arr1)=>{
            Int32ToByte(parseInt(cs2, 2), (arr2)=>{
                arr1 = arr1.concat(arr2);
                callback(arr1);
            });
        });
    } else {
        Int32ToByte(value, (arr)=>{
            callback(arr);
        });
    }
}

// Int32ToByte change Int32 num to byte array
function Int32ToByte(value, callback) {
    let byteArr = [];
    for (let i = 16; i >= 0; i = i - 16) {
        if((value >> i & 0xffff) != 0) {
            byteArr.push(value >> i & 0xffff);
        }
    }
    callback(byteArr);
}

// switchRegister reverse the order of array
function switchRegister(data, callback) {
    let switchData = [];
    for (let i = 0; i < data.length/2; i++) {
        switchData[i] = data[data.length-i-1];
        switchData[data.length-i-1] = data[i];
    }
    callback(switchData)
}

// switchByte exchange lower and higher byte value of two byte data
function switchByte(data, callback){
    let switchData = [];
    let InternalData = [];
    for (let i = 0; i < data.length; i++){
        InternalData[0] = data[i] & 0xff;
        InternalData[1] = data[i] >> 8 & 0xff;
        byteArrayToInt(InternalData, (bitarr)=>{
            switchData[i] = bitarr;
        });
    }
    callback(switchData)
}

module.exports = {bitArrayToInt, byteArrayToInt, IntToByteArray, switchRegister, switchByte, twoByteArrayToInt};
