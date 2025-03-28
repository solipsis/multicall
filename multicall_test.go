package multicall

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestBalanceOf(t *testing.T) {

	client, err := ethclient.Dial("https://ethereum-rpc.publicnode.com")
	if err != nil {
		t.Fatal(err)
	}

	weeth := common.HexToAddress("0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee")
	users := []common.Address{
		common.HexToAddress("0xcd2eb13D6831d4602D80E5db9230A57596CDCA63"),
		common.HexToAddress("0xBdfa7b7893081B35Fb54027489e2Bc7A38275129"),
		common.HexToAddress("0xa3A7B6F88361F48403514059F1F16C8E78d60EeC"),
	}

	mc := &MulticallClient{Client: client}
	balances, err := BalanceOf(mc, users, weeth, &bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	for idx, balance := range balances {
		fmt.Printf("User: %s, balance: %s\n", users[idx], balance)
	}
}
