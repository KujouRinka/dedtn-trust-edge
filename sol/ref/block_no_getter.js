const fs = require("fs");
const readline = require('readline')
const Web3 = require('web3');
const global_web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));
const areaA_web3 = new Web3(new Web3.providers.HttpProvider("http://[IPアドレス]:8555"));
const areaB_web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8555"));

/** 取得したトランザクションハッシュにコンマをつける **/
function konma(inp, outp) {
    const rs = fs.createReadStream(inp);
    const ws = fs.createWriteStream(outp);

    const rl = readline.createInterface({
        input: rs,
        output: ws
    });

    rl.on('line', (lineString) => {
        ws.write("\"" + lineString + '\", ');
    });
}

/** ブロック番号に変換 **/
function getBlockNum(input, output, web) {
    for (let i = 0; i < input.length; i++) {
        new Promise((resolve) => {
            resolve(web.eth.getTransaction(input[i]));
        })
            .then((tx) => {
                fs.appendFile(output, tx.blockNumber + "\n", (err) => {
                });
            });
    }
}

// 実行///////////////////////////
// konma('data/global_TX.txt', 'data/global_konma.txt');
// konma('data/areaA_TX.txt', 'data/areaA_konma.txt');
// konma('data/areaB_TX.txt', 'data/areaB_konma.txt');

// konma関数実行後の各ファイル内を各配列に貼り付ける
const areaA = [];
const areaB = [];
const global = [];


//getBlockNum(global, 'data/global_BlockNum.txt', global_web3);
//getBlockNum(areaA, 'data/areaA_BlockNum.txt', areaA_web3);
//getBlockNum(areaB, 'data/areaB_BlockNum.txt', areaB_web3);