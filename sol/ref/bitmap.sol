// SPDX−License−Identifier: GPL−3.0
pragma solidity >=0.4.16 <0.9.0;

contract TAManagement {
    struct Area {
        string AreaName; //エリア名 ex)Kanazawa, Tokyo
        address RAaddress; //エリアを管理するTA
    }

    struct Vehicle {
        address acno;
        address Area;
    }

    Vehicle[] Vehicles;
    Area[] Areas;

    /**新規車の登録 **/
    function NewVehicle(address Vad) public {
        //登録済みではないかチェックループ
        for (uint i = 0; i < Vehicles.length; i++) {
            require(Vehicles[i].acno != Vad, "You have already registered this vehicle");
        }
        Vehicles.push(Vehicle(Vad, msg.sender));
    }

    /**エリアの更新 **/
    function UpdateArea(address acno) public {
        for (uint i = 0; i < Vehicles.length; i++) {
            if (Vehicles[i].acno == acno)//Index取得
                Vehicles[i].Area = msg.sender;
        }
    }

    /**所属エリア参照 **/
    function ansRAad(address Vad) public view returns (address) {
        address tmp;
        for (uint i = 0; i < Vehicles.length; i++) {
            if (Vehicles[i].acno == Vad)
                tmp = Vehicles[i].Area;
        }
        require(tmp != address(0), "cannot find the vehicle in all system");
        return tmp;
    }

    /**新規エリアの登録 **/
    function NewArea(string memory AreaName) /*only RA*/ public {
        if (Areas.length != 0) {
            //すでに登録済みのAreaではないかチェック
            for (uint i = 0; i < Areas.length; i++) {
                require(Areas[i].RAaddress != msg.sender, "This area already has registered");
                require(!compare(Areas[i].AreaName, AreaName), "This AreaNamehas used by other area, so you have to change your AreaName");
            }
        }
        Areas.push(Area(AreaName, msg.sender));
    }

    /** Areas配列内容チェック用 **/
    function AreaCheck(uint Index) public view returns (string memory, address){
        return (Areas[Index].AreaName, Areas[Index].RAaddress);
    }

    /** Vehicles配列内容チェック用 **/
    function VehicleCheck(uint VehicleIndex) public view returns (address, address){
        Vehicle storage v = Vehicles[VehicleIndex];
        return (v.acno, v.Area);
    }

    /**文字列比較用 **/
    function compare(string memory a, string memory b) pure internal returns (bool) {
        if (bytes(a).length != bytes(b).length) {
            return false;
        } else {
            return keccak256(abi.encodePacked(a)) == keccak256(abi.encodePacked(b));
        }
    }
}