package multicall

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gadzefinance/topup/src/contracts"
)

// BalanceOf is a multicall batched call for any contract that implements the following solidity interface
// ERC20 function balanceOf(address addr) external returns (uint256)
func BalanceOf(mc *MulticallClient, addresses []common.Address, target common.Address, callOpts *bind.CallOpts) ([]*big.Int, error) {
	erc20ABI, _ := contracts.ERC20MetaData.GetAbi()
	method := erc20ABI.Methods["balanceOf"]

	// how to interpret the return value of balanceOf
	parseFunc := func(decoded []any) *big.Int {
		if len(decoded) == 0 {
			return big.NewInt(0)
		}
		return *abi.ConvertType(decoded[0], new(*big.Int)).(**big.Int)
	}

	// split into smaller queries if over maximum batch and fetch concurrently
	maxBatchSize := 5000
	balances, err := Call(mc, callOpts, method, target, addresses, parseFunc, maxBatchSize)
	if err != nil {
		return nil, fmt.Errorf("fetching balances via multicall: %w", err)
	}

	return balances, err
}

// TODO: add other ERC20 methods
