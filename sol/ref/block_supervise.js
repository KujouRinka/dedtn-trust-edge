const Web3 = require('web3');
const global_web3 = new Web3("ws://localhost:8545");
const local_web3 = new Web3("ws://localhost:8546");
const fs = require('fs');


/** チェーンno.15を監視 **/
const Blocks15 = global_web3.eth.subscribe("newBlockHeaders");
Blocks15.on("data", (blockHeader) => {
    let data = "block: " + blockHeader["number"] + " − ts: " + blockHeader["timestamp"] + " − received at: " + Math.floor(new Date() / 1000) + " − dif: " + blockHeader["difficulty"];
    fs.appendFile("data/global_block.txt", data + "\n", (err) => {
    });
});

/** チェーンAまたはチェーンBを監視 **/
const Blocks10 = local_web3.eth.subscribe("newBlockHeaders");
Blocks10.on("data", (blockHeader) => {
    let data10 = "block: " + blockHeader["number"] + " − ts: " + blockHeader["timestamp"] + " − received at: " + Math.floor(new Date() / 1000) + " − dif: " + blockHeader["difficulty"];
    fs.appendFile("data/areaB_block.txt", data10 + "\n", (err) => {
    });
});