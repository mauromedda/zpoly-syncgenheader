# zpoly-syncgenheader

This tool is a simplified version of https://github.com/polynetwork/poly-io-test/tree/035b7fadee297e6e1b5a0b3dcde80f22442d8fb1/cmd/tools/run.go

Provides the possibility to execute the SyncGenesisHeader for a given
```
txBlockAndDsComm := TxBlockAndDsComm{
		TxBlock: txBlock,
		DsBlock: dsBlock,
		DsComm:  dsComm,
	}
```

On testnet we don't have the possibility to specify a fixed TX block because the Zilliqa API GetMinerInfo
will not return a dsComm required to compose the `txBlockAndDsComm` used as input argument
of `poly.Native.Hs.SyncGenesisHeader`.


# How to run the script

1. checkout the repository https://github.com/polynetwork/poly-io-test/

```
git clone https://github.com/polynetwork/poly-io-test
```

2. Override the file poly-io-test/cmd/tools/run.go with the run.go provided in `zpoly-syncgenheader.zip`

```
unzip zpoly-syncgenheader.zip

cd zpoly-syncgenheader && \
cp -p run.go /path/to/poly-io-test/cmd/tools/run.go

```

3. Build the go program

```
cd /path/to/poly-io-test/cmd/tools/
go build
```

4. Create a suitable configuration file for the zilliqa testnet
** you need a poly network wallet ***

```
{
	"ZilChainID": 333,
	"ZilURL": "https://dev-api.zilliqa.com",
	"ZilPrivateKey": "",
	"GasPrice": 2500,
	"GasLimit": 30000000,
	"ReportInterval": 60,
	"BatchTxNum": 1,
	"BatchInterval": 1,
	"TxNumPerBatch": 1,
	"RchainJsonRpcAddress": "http://beta1.poly.network:20336",
	"RCWallet": "./poly.wallet",
	"RCWalletPwd": "polywalletpwd"
}
```

# Verify the configuration
./tools -tool get_poly_consensus

# run ZILSyncGenesysHeader from the latest testnet txblock

./tools -tool sync_genesis_header -ppwds <polywalletpwd> -pwallets ./poly.wallet
