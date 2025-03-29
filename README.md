# Multicall

A dead simple multicall client that batches and concurrently executes provided calls


        // connect to an rpc provider
        client, _ := ethclient.Dial("https://ethereum-rpc.publicnode.com")

        // create the multicall client
        mc := &MulticallClient{Client: client}

        // pick a method and contract to multicall
        erc20ABI, _ := contracts.ERC20MetaData.GetAbi()
        method := erc20ABI.Methods["balanceOf"]
        target := common.HexToAddress("0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee")

        // define how to interpret each return value
        parseFunc := func(decoded []any) *big.Int {
            if len(decoded) == 0 {
                return big.NewInt(0)
            }
            return *abi.ConvertType(decoded[0], new(*big.Int)).(**big.Int)
        }

        // execute the multicall
        maxBatchSize := 1000 // maximum number of subcalls to pack into each multicall
        balances, _ := Call(mc, &bind.CallOpts{}, method, target, addresses, parseFunc, maxBatchSize)



