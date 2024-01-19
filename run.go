/*
* Copyright (C) 2020 The poly network Authors
* This file is part of The poly network library.
*
* The poly network is free software: you can redistribute it and/or modify
* it under the terms of the GNU Lesser General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The poly network is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Lesser General Public License for more details.
* You should have received a copy of the GNU Lesser General Public License
* along with The poly network . If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"os"
	"strconv"
	"strings"

	"github.com/Zilliqa/gozilliqa-sdk/core"
	"github.com/Zilliqa/gozilliqa-sdk/provider"
	zilutil "github.com/Zilliqa/gozilliqa-sdk/util"

	poly_go_sdk "github.com/polynetwork/poly-go-sdk"

	"github.com/polynetwork/poly-io-test/chains/btc"
	"github.com/polynetwork/poly-io-test/config"
	"github.com/polynetwork/poly-io-test/log"
	"github.com/polynetwork/poly-io-test/testcase"
	"github.com/polynetwork/poly/common"
	"github.com/polynetwork/poly/native/service/governance/node_manager"
	"github.com/polynetwork/poly/native/service/utils"
)

var (
	tool                                                                  string
	toolConf                                                              string
	pWalletFiles                                                          string
	pPwds                                                                 string
	oWalletFiles                                                          string
	oPwds                                                                 string
	newWallet                                                             string
	newPwd                                                                string
	amt                                                                   int64
	keyFile                                                               string
	stateFile                                                             string
	id                                                                    uint64
	blockMsgDelay, hashMsgDelay, peerHandshakeTimeout, maxBlockChangeView uint64
	rootca                                                                string
	chainId                                                               uint64
	fabricRelayerTy                                                       uint64
	neo3StateValidators                                                   string
)

func init() {
	flag.StringVar(&tool, "tool", "", "choose a tool to run")
	flag.StringVar(&toolConf, "conf", "./config.json", "configuration file path")
	flag.StringVar(&pWalletFiles, "pwallets", "", "poly wallet files sep by ','")
	flag.StringVar(&pPwds, "ppwds", "", "poly pwd for every wallet, sep by ','")
	flag.Uint64Var(&blockMsgDelay, "blk_msg_delay", 5000, "")
	flag.Uint64Var(&hashMsgDelay, "hash_msg_delay", 5000, "")
	flag.Uint64Var(&peerHandshakeTimeout, "peer_handshake_timeout", 10, "")
	flag.Uint64Var(&maxBlockChangeView, "max_blk_change_view", 10000, "")
	flag.StringVar(&rootca, "rootca", "", "file path for root CA")
	flag.Uint64Var(&chainId, "chainid", 333, "default 333 means zilliqa")
	flag.Parse()
}

func main() {
	log.InitLog(2, os.Stdout)

	err := config.DefConfig.Init(toolConf)
	if err != nil {
		panic(err)
	}
	poly := poly_go_sdk.NewPolySdk()
	if err := btc.SetUpPoly(poly, config.DefConfig.RchainJsonRpcAddress); err != nil {
		panic(err)
	}

	switch tool {

	case "sync_genesis_header":
		wArr := strings.Split(pWalletFiles, ",")
		pArr := strings.Split(pPwds, ",")

		accArr := make([]*poly_go_sdk.Account, len(wArr))
		for i, v := range wArr {
			accArr[i], err = btc.GetAccountByPassword(poly, v, []byte(pArr[i]))
			if err != nil {
				panic(fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, wArr[i], pArr[i]))
			}
		}

		switch chainId {

		case config.DefConfig.ZilChainID:
			SyncZILGenesisHeader(poly, accArr)
		}
	case "get_poly_config":
		GetPolyConfig(poly)

	case "get_poly_consensus":
		GetPolyConsensusInfo(poly)
	}
}

func getPolyAccounts(poly *poly_go_sdk.PolySdk) []*poly_go_sdk.Account {
	wArr := strings.Split(pWalletFiles, ",")
	pArr := strings.Split(pPwds, ",")
	accArr := make([]*poly_go_sdk.Account, len(wArr))
	var err error
	for i, v := range wArr {
		accArr[i], err = btc.GetAccountByPassword(poly, v, []byte(pArr[i]))
		if err != nil {
			panic(fmt.Errorf("failed to decode no%d wallet %s with pwd %s", i, wArr[i], pArr[i]))
		}
	}
	return accArr
}

func SyncZILGenesisHeader(poly *poly_go_sdk.PolySdk, accArr []*poly_go_sdk.Account) {
	type TxBlockAndDsComm struct {
		TxBlock *core.TxBlock
		DsBlock *core.DsBlock
		DsComm  []core.PairOfNode
	}

	zilSdk := provider.NewProvider(config.DefConfig.ZilURL)
	// ON TESTNET it gets the currentDScomm. The getMiner info returns the an empty dscommittee
	// for a previous DSBlock num
	initDsComm, _ := zilSdk.GetCurrentDSComm()
	// as its name suggest, the tx epoch is actually a future tx block
	// zilliqa side has this limitation to avoid some risk that no tx block got mined yet
	nextTxEpoch, _ := strconv.ParseUint(initDsComm.CurrentTxEpoch, 10, 64)
	fmt.Printf("current tx block number is %s, ds block number is %s, number of ds guard is: %d\n", initDsComm.CurrentTxEpoch, initDsComm.CurrentDSEpoch, initDsComm.NumOfDSGuard)

	for {
		latestTxBlock, _ := zilSdk.GetLatestTxBlock()
		fmt.Println("wait current tx block got generated")
		latestTxBlockNum, _ := strconv.ParseUint(latestTxBlock.Header.BlockNum, 10, 64)
		fmt.Printf("latest tx block num is: %d, current tx block num is: %d", latestTxBlockNum, nextTxEpoch)
		if latestTxBlockNum >= nextTxEpoch {
			break
		}
		time.Sleep(time.Second * 20)
	}

	_, err := zilSdk.GetNetworkId()
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader failed: %s", err.Error()))
	}

	var dsComm []core.PairOfNode
	for _, ds := range initDsComm.DSComm {
		dsComm = append(dsComm, core.PairOfNode{
			PubKey: ds,
		})
	}
	dsBlockT, err := zilSdk.GetDsBlockVerbose(initDsComm.CurrentDSEpoch)
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader get ds block %s failed: %s", initDsComm.CurrentDSEpoch, err.Error()))
	}
	dsBlock := core.NewDsBlockFromDsBlockT(dsBlockT)

	txBlockT, err := zilSdk.GetTxBlockVerbose(initDsComm.CurrentTxEpoch)
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader get tx block %s failed: %s", initDsComm.CurrentTxEpoch, err.Error()))
	}

	txBlock := core.NewTxBlockFromTxBlockT(txBlockT)

	txBlockAndDsComm := TxBlockAndDsComm{
		TxBlock: txBlock,
		DsBlock: dsBlock,
		DsComm:  dsComm,
	}

	raw, err := json.Marshal(txBlockAndDsComm)
	if err != nil {
		panic(fmt.Errorf("SyncZILGenesisHeader marshal genesis info failed: %s", err.Error()))
	}
	fmt.Println(raw)
	// sync zilliqa genesis info onto polynetwork
	txhash, err := poly.Native.Hs.SyncGenesisHeader(config.DefConfig.ZilChainID, raw, accArr)
	if err != nil {
		if strings.Contains(err.Error(), "had been initialized") {
			log.Info("zil already synced")
		} else {
			panic(fmt.Errorf("SyncZILGenesisHeader failed: %v", err))
		}
	} else {
		testcase.WaitPolyTx(txhash, poly)
		log.Infof("successful to sync zil genesis header, ds block: %s, tx block: %s, ds comm: %+v\n", zilutil.EncodeHex(dsBlock.BlockHash[:]), zilutil.EncodeHex(txBlock.BlockHash[:]), dsComm)
	}

}

func GetPolyConfig(poly *poly_go_sdk.PolySdk) {
	raw, err := poly.GetStorage(utils.NodeManagerContractAddress.ToHexString(), []byte(node_manager.VBFT_CONFIG))
	if err != nil {
		panic(err)
	}
	conf := &node_manager.Configuration{}
	if err = conf.Deserialization(common.NewZeroCopySource(raw)); err != nil {
		panic(err)
	}
	log.Infof("poly config: (blockMsgDelay: %d, hashMsgDelay: %d, peerHandshakeTimeout: %d, maxBlockChangeView: %d)",
		conf.BlockMsgDelay, conf.HashMsgDelay, conf.PeerHandshakeTimeout, conf.MaxBlockChangeView)
}

func GetPolyConsensusInfo(poly *poly_go_sdk.PolySdk) {
	storeBs, err := poly.GetStorage(utils.NodeManagerContractAddress.ToHexString(), []byte(node_manager.GOVERNANCE_VIEW))
	if err != nil {
		panic(err)
	}
	source := common.NewZeroCopySource(storeBs)
	gv := new(node_manager.GovernanceView)
	if err := gv.Deserialization(source); err != nil {
		panic(err)
	}

	raw, err := poly.GetStorage(utils.NodeManagerContractAddress.ToHexString(),
		append([]byte(node_manager.PEER_POOL), utils.GetUint32Bytes(gv.View)...))
	if err != nil {
		panic(err)
	}
	m := &node_manager.PeerPoolMap{
		PeerPoolMap: make(map[string]*node_manager.PeerPoolItem),
	}
	if err := m.Deserialization(common.NewZeroCopySource(raw)); err != nil {
		panic(err)
	}
	str := ""
	for _, v := range m.PeerPoolMap {
		str += fmt.Sprintf("[ index: %d, address: %s, pubk: %s, status: %d ]\n",
			v.Index, v.Address.ToBase58(), v.PeerPubkey, v.Status)
	}

	log.Infof("get consensus info of poly: { view: %d, len_nodes: %d, info: \n%s }", gv.View, len(m.PeerPoolMap), str)
}
