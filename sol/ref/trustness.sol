// SPDX−License−Identifier: GPL−3.0
pragma solidity >=0.4.16 <0.9.0;

contract VehicleManagement {

/** Vehicle Structure **/
    struct Vehicle_Register {
        int TV;
        uint256 credit_score;
        address acno;
        bool Revoked;
        bool is_VEH_intelligent;
        uint claimed;
        uint claim_limit;
        uint time;
        address Area;
    }

    /** RSU − Road Side Unit Structure **/
    struct RSU {
        address RSUId;
        string name;
        uint256 s_uid;
        uint256 reg_v_id;
    }

    /** Personal Transaction Register specific to each particular Vehicle **/
    struct Personal_Transaction_Register {
        bool submitted;
        bool isrewardReceived;
    }

    struct Score_Status_Register {
        uint score; //あるsessionにおいて、ある車が疑わしいとされた報告の数
        bool verificationStatus;
    }

    struct Session_Register {
        uint count;
        address registered_address;
        address alarmer;
        bool enrolled;
    }

    struct Vehicles_in_Session_Register {
        address registered_address;
        address doubty_vehicle;
    }

    struct Claimable_Reward {
        address session;
        address suspected_vehicle;
    }

    uint256 TA_Round = 0;

    address[] Vehicleaddress;
    address[] RevocationList;
    address[] RegisteredSession;
    mapping(address => mapping(address => mapping(address =>
    Personal_Transaction_Register))) PTR;
    mapping(address => mapping(address => Score_Status_Register)) SSR;

    mapping(address => mapping(uint256 => Vehicles_in_Session_Register)) VISR;
    mapping(address => Vehicle_Register) VR;
    mapping(address => Session_Register) SESSION_REGISTER;
    mapping(address => mapping(uint => Claimable_Reward)) CR;
    /** Vehicle Registration, Only performed by the RA **/
    function configureVEH(address _acno) public returns (address, address) {
        VR[_acno].acno = _acno;
        VR[_acno].TV = 10;
        VR[_acno].credit_score = 100;
        VR[_acno].Revoked = false;
        VR[_acno].claim_limit = 0;
        VR[_acno].claimed = 0;
        VR[_acno].time = block.timestamp;
        VR[_acno].Area = msg.sender;
        VR[_acno].is_VEH_intelligent = true;

        return (_acno, msg.sender);
    }

    /** 悪意のある車の報告**/
    //sessionの識別はウォレットアドレス
    function reportMisbehavior(address session, address suspected_vehicle) public {
        //すでにsessionリストに含まれていないか．mappingだからenrolledの初期値false
        //はじめにこのsession報告するとalarmerになる
        if (SESSION_REGISTER[session].enrolled != true) {
            //sessionをRegisteredSession配列に入れる
            RegisteredSession.push(session);
            //はじめにこのsessionを発見したのがmsg.sender
            SESSION_REGISTER[session].alarmer = msg.sender;
            SESSION_REGISTER[session].enrolled = true;
        }
        //報告数+ 1
        SESSION_REGISTER[session].count = SESSION_REGISTER[session].count + 1;
        //sessionの[報告数]番目を報告したアドレスはmsg.sender
        VISR[session][SESSION_REGISTER[session].count].registered_address = msg.sender;
        VISR[session][SESSION_REGISTER[session].count].doubty_vehicle =
                    suspected_vehicle;

        //msg.senderがsessionにおけるsuspected_vehicleについてのTxを提出済みか
        if (PTR[msg.sender][session][suspected_vehicle].submitted == false) {
            //sessionにおける疑わしい車がsuspected_vehicleである報告をカウント
            SSR[session][suspected_vehicle].score = SSR[session][suspected_vehicle].score + 1;
            //提出済みにする
            PTR[msg.sender][session][suspected_vehicle].submitted = true;
            //報酬はまだもらってない
            PTR[msg.sender][session][suspected_vehicle].isrewardReceived = false;
            //claim_limitはmsg.senderが報告したclaim(sessionにおけるsuspected_vehicle)の数
            CR[msg.sender][VR[msg.sender].claim_limit].session = session;
            CR[msg.sender][VR[msg.sender].claim_limit].suspected_vehicle = suspected_vehicle;
            //報告完了→claim_limitが1増える
            VR[msg.sender].claim_limit = VR[msg.sender].claim_limit + 1;
        } else {
            //Tx提出済みだったら何もしなくていい
            revert();
        }
    }

    /** 報告の集計と悪意のある車の認定**/
    //TA_Round番目のsessionについて
    function TACheck() public {
        //TA_Roundの初期値0．RegisterdSession配列はsessionのリスト
        address session = RegisteredSession[TA_Round];
        //同一sessionについての報告全てをチェック
        for (uint i = 1; i <= SESSION_REGISTER[session].count; i++) {
            //sessionのi番目の報告で疑わしいとされた車のアドレス
            address suspected_vehicle = VISR[session][i].doubty_vehicle;
            //SSR: sessionのi番目の報告で疑わしいとされたsuspected_vehicleが
            //その他の車からもsuspected_vehicleとして報告された数
            //SESSION_REGISTER: sessionが報告された回数
            //つまりあるsessionにおいて、同一の車が疑わしいという報告が
            //全体の報告数の半分以上だったら
            //有罪！悪意のある車と認定(ちょうど半分だったら無罪)
            if (SSR[session][suspected_vehicle].score > SESSION_REGISTER[session].count / 2) {
                //TVから1引く
                VR[suspected_vehicle].TV = VR[suspected_vehicle].TV - 1;
                //1分間IsBlocked修飾子で引っかかる
                VR[suspected_vehicle].time = VR[suspected_vehicle].time + 1 minutes;
                // TVが0を下回ったらrevokeに
                if (VR[suspected_vehicle].TV < 0 && VR[suspected_vehicle].Revoked == false/*まだRevokedされてなかったら*/) {
                    //Revocation配列に入れる
                    RevocationList.push(suspected_vehicle);
                    //revoke済に
                    VR[suspected_vehicle].Revoked = true;
                }
                // sessionにおけるsuspected_vehicle処理済み(罰した)
                SSR[session][suspected_vehicle].verificationStatus = true;
            }
        }
        TA_Round = TA_Round + 1;
    }

    /** 報酬を与える**/
    //呼び出すのは報告した車
    function claimReward() public {
        address session = CR[msg.sender][VR[msg.sender].claimed].session;
        address suspected_vehicle = CR[msg.sender][VR[msg.sender].claimed].suspected_vehicle;

        //まだmsg.senderがsessionのsuspected_vehicle報告
        //においてrewardもらってない
        if (PTR[msg.sender][session][suspected_vehicle].isrewardReceived == false) {
            // TACheckでもうすでにsessionにおけるsuspected_vehicleを罰し済みか
            // TACheckで有罪判決が出なかったmsg.senderはこのif文に入れない
            if (SSR[session][suspected_vehicle].verificationStatus == true) {
                //msg.senderがsessionにおけるsuspected_vehicleを
                //報告済みか(ReportMisbehavior呼び出し済みか)
                if (SSR[session][suspected_vehicle].verificationStatus == PTR[msg.sender][session][suspected_vehicle].submitted) {
                    VR[msg.sender].credit_score = VR[msg.sender].credit_score + 5;
                    //報告数1個上げる
                    VR[msg.sender].claimed = VR[msg.sender].claimed + 1;

                    //msg.senderがsession発見者だったら
                    if (SESSION_REGISTER[session].alarmer == msg.sender) {
                        //credit_scoreさらに2個上げる
                        VR[msg.sender].credit_score = VR[msg.sender].credit_score + 2;
                    }
                    //sessionにおいてsuspected_vehicleを報告したmsg. senderを報酬受け取り済みに
                    PTR[msg.sender][session][suspected_vehicle].isrewardReceived = true;
                }
            }
        } else {
            revert();
        }
    }

    /** 移動報告**/
    function Enter(address fromArea) public view returns (address Vad, address RAad) {
        return (msg.sender, fromArea);
    }

    /** 信頼性の提供(エリア移動時) **/
    function Exit2(address exiVEH) public view returns (int TV, uint credit_score, address acno, bool Revoked, bool is_VEH_intelligent, uint claimed, uint claim_limit, uint time) {
        Vehicle_Register memory v = VR[exiVEH];
        return (v.TV, v.credit_score, v.acno, v.Revoked, v.is_VEH_intelligent, v.claimed, v.claim_limit, v.time);
    }

    /** 信頼性の追加**/
    function regVEH(int TV, uint credit_score, address acno, bool Revoked, bool is_VEH_intelligent, uint claimed, uint claim_limit, uint time) public {
        VR[acno] = Vehicle_Register({TV: TV, credit_score: credit_score, acno: acno,
            Revoked: Revoked, is_VEH_intelligent: is_VEH_intelligent, claimed: claimed,
            claim_limit: claim_limit, time: time, Area: msg.sender});

        VR[acno].Area = msg.sender;
    }

    /** 信頼性の消去**/
    function Exit3(address exiVEH) public {
        VR[exiVEH] = Vehicle_Register({
            TV: - 999,
            credit_score: 1000,
            acno: address(0),
            Revoked: true,
            is_VEH_intelligent: true,
            claimed: 100,
            claim_limit: 100,
            time: 100, Area: address(0)
        });
    }

    /** 信頼性の要求(別エリアの車の信頼性取得) **/
    function askTV(address Vad) public view returns (int256){
        if (msg.sender == VR[Vad].Area) {
            return (VR[Vad].TV/*, msg.sender*/);
        }
        else {
            return (- 999);
        }
    }

    /** 以下チェック用**/
    function CRStatus() view public returns (bool, bool, bool, bool){
        address session = CR[msg.sender][VR[msg.sender].claimed].session;
        address suspected_vehicle = CR[msg.sender][VR[msg.sender].claimed].suspected_vehicle;

        return (
            PTR[msg.sender][session][suspected_vehicle].isrewardReceived,
            SSR[session][suspected_vehicle].verificationStatus,
            PTR[msg.sender][session][suspected_vehicle].submitted,
            SESSION_REGISTER[session].alarmer == msg.sender
        );
    }

    function Nakami() view public returns (address, address, uint){
        address CRsession = CR[msg.sender][VR[msg.sender].claimed].session;
        address CRsuspected_vehicle = CR[msg.sender][VR[msg.sender].claimed].suspected_vehicle;

        return (CRsession, CRsuspected_vehicle, VR[msg.sender].claimed);
    }

    function checkmyTVC() view public returns (int, uint){
        return (VR[msg.sender].TV, VR[msg.sender].credit_score);
    }

    function CheckmyClaims() view public returns (string memory){
        if (VR[msg.sender].claimed == VR[msg.sender].claim_limit) {
            return ("No more pending Claims");
        }

        if (SSR[CR[msg.sender][VR[msg.sender].claimed].session][CR[msg.sender][VR[msg.sender].claimed].suspected_vehicle].verificationStatus != true) {
            return ("Claim not yet verified by TA");
        }
        else {
            return ("You have got pending Claims. Check Pending Claim Details for more info");
        }
    }
}