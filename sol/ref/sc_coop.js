const fs = require("fs");
const Web3 = require('web3');
const areaA_global_web3 = new Web3(new Web3.providers.HttpProvider("http ://[IPアドレス]:8545"));
const areaA_local_web3 = new Web3(new Web3.providers.HttpProvider("http://[IPアドレス]:8555"));
const areaB_global_web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8545"));
const areaB_local_web3 = new Web3(new Web3.providers.HttpProvider("http://localhost:8555"));

// ファイル読み込み//////////////////////////////////
var global_bin, global_abi, local_bin, local_abi;
local_bin = '0x123...' //信頼性評価用SCのバイナリ
local_abi = [] //信頼性評価用SCのabi
global_bin = '0x456...' //マッピング用SCのバイナリ
global_abi = [] //マッピング用SCのabi

//RAと車のアドレス//////////////////////////////////
const areaA_RA = "0x776...";
const areaA_VEH = ["0x2e6...", "0xa41c...", "0x7cd...", "0x8d0...", "0x35a...", "0xd12..."];
const areaB_RA = "0x64b...";
const areaB_VEH = ["0xd4f...", "0x028...", "0x21f...", "0x7fb...", "0x7e5...", "0xb02..."];

// コントラクトクラス定義//////////////////////////////
const areaA_global_contract = new areaA_global_web3.eth.Contract(global_abi, "0xFd4.../*コントラクトアドレス*/", {
    from: areaA_RA,
    data: global_bin
});
const areaA_local_contract = new areaA_local_web3.eth.Contract(local_abi, "0x2ec.../*コントラクトアドレス*/", {
    from: areaA_RA, data: local_bin
});
const areaB_global_contract = new areaB_global_web3.eth.Contract(global_abi, "0xFd4.../*コントラクトアドレス*/", {
    from: areaB_RA,
    data: global_bin
});
const areaB_local_contract = new areaB_local_web3.eth.Contract(local_abi, "0x9f9.../*コントラクトアドレス*/", {
    from: areaB_RA,
    data: local_bin
});


const areaA_global = areaA_global_contract.methods;
const areaA_local = areaA_local_contract.methods;
const areaB_global = areaB_global_contract.methods;
const areaB_local = areaB_local_contract.methods;


//関数////////////////////////////////////////
/** デプロイ **/
function deploy(RAaddress, Contract) {
    Contract.deploy()
        .send({
            from: RAaddress
        })
        .then(function (instance) {
            console.log("contract: " + instance.options.address);
        });
}

/** 新規エリアの登録 **/
function NewArea(RAaddress, AreaName, Contract) {
    Contract.methods.NewArea(AreaName).send({from: RAaddress});
}

/** 新規車の登録 **/
function RegVehicle(global, local, Vads, RAaddress) {
    for (let i = 0; i < Vads.length; i++) {
        // global.NewVehicle(Vads[i]).send({from: RAaddress});
        local.configureVEH(Vads[i]).send({from: RAaddress});
    }
}

/** TV確認 **/
function TVkakunin(local, Vehicle) {
    for (let i = 0; i < Vehicle.length; i++) {
        local.checkmyTVC().call({from: Vehicle[i]})
            .then(console.log);
    }
}

/** 一台の車がエリアの境界を通過するときの一連の処理 **/
function transVEH(re, Vehicle, fromArea, toArea, bet, fromSC, toSC, fromtxt, totxt) {
    let p = re;
    return new Promise(resolve => {
        //車→RA移動報告
        toSC.Enter(fromArea).call({from: Vehicle})
            .then((EnterResult) => {
                let pro1 = new Promise((resolve) => {
                    //エリア更新
                    bet.UpdateArea(EnterResult.Vad).send({from: toArea})
                        .once('transactionHash', function (hash) {
                            fs.appendFile("data/global_TX.txt", hash + '\n', (err) => {
                            });
                        })
                        .then(() => {
                            resolve();
                        });
                });
                let pro2 = new Promise((resolve) => {
                    //信頼性の提供
                    fromSC.Exit2(EnterResult.Vad).call({from: fromArea})
                        .then((v) => {
                            //信頼性の追加
                            toSC.regVEH(v.TV, v.credit_score, v.acno, v.Revoked, v.is_VEH_intelligent, v.claimed, v.claim_limit, v.time).send({from: toArea})
                                .once('transactionHash', function (hash) {
                                    fs.appendFile(totxt, hash + '\n', (err) => {
                                    });
                                })
                                .then(() => {
                                    //信頼性の削除
                                    fromSC.Exit3(EnterResult.Vad).send({from: fromArea})
                                        .once('transactionHash', function (hash) {
                                            fs.appendFile(fromtxt, hash + '\n', (err) => {
                                            });
                                        })
                                        .then(() => {
                                            resolve();
                                        })
                                });
                        });
                });
                Promise.all([pro1, pro2])
                    .then(() => {
                        resolve(p);
                    });
            });
    });
}

/** 一台の車がエリアの境界を2回通過(1往復) **/
function oneRound(index, Vehicle, firstRA, secondRA, first_global, second_global, first_local, second_local, firstData, secondData) {
    return new Promise((resolve) => {
        transVEH(1, Vehicle, firstRA, secondRA, second_global, first_local, second_local, firstData, secondData)
            .then(async (re) => {
                await transVEH(re, Vehicle, secondRA, firstRA, first_global, second_local, first_local, secondData, firstData);
                index++;
                resolve(index);
            });
    })
}

/**車がnum*2回往復する **/
async function manyTrans(num, Vehicle, firstRA, secondRA, first_global, second_global, first_local, second_local, firstData, secondData) {
    for (let i = 0; i < num; i++) {
        await oneRound(1, Vehicle, firstRA, secondRA, first_global, second_global, first_local, second_local, firstData, secondData);
    }
}

/** n_vehicle台の車がn_trans*2回往復する **/
function manymanyTrans(n_trans, n_vehicle) {
    for (let l = 0; l < n_vehicle; l++) {//lは車の台数
        //A→B→A//引数一つめが往復数
        manyTrans(n_trans, areaA_VEH[l], areaA_RA, areaB_RA, areaA_global, areaB_global, areaA_local, areaB_local, "data/areaA_TX.txt", "data/areaB_TX.txt");
        //B→A→B
        manyTrans(n_trans, areaB_VEH[l], areaB_RA, areaA_RA, areaB_global, areaA_global, areaB_local, areaA_local, "data/areaB_TX.txt", "data/areaA_TX.txt");
    }
}


// 実行用////////////////////////////////////
// deploy(areaB_RA, areaB_global_contract);
// deploy(areaB_RA, areaB_local_contract);
// deploy(areaA_RA, areaA_local_contract);
// deploy(areaC_RA, areaC_local_contract);
// NewArea(areaA_RA, "Nagano", areaA_global_contract);
// NewArea(areaB_RA, "Toyama", areaB_global_contract);
// // NewArea(areaC_RA, "Nigata", areaC_global_contract);
// RegVehicle(areaA_global, areaA_local, areaA_VEH, areaA_RA);
// RegVehicle(areaB_global, areaB_local, areaB_VEH, areaB_RA);
// RegVehicle(areaC_global, areaC_local, areaC_VEH, areaC_RA);
// 一旦確認
// TVkakunin(areaB_local, areaB_VEH);
// areaB_global.AreaCheck(1).call()
// .then(console.log);
// manymanyTrans(50, 6);