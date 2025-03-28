package multicall

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/sync/errgroup"
)

var (
	zeroAddress               = common.HexToAddress("0x0000000000000000000000000000000000000000")
	DefaultMulticallAddr      = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	DefaultMaxConcurrentCalls = -1 // no limit
)

type MulticallClient struct {
	Client             *ethclient.Client
	MulticallAddress   common.Address
	MaxConcurrentCalls int
}

// Call splits the provided call data into batches of at most "maxBatchSize" and executes them concurrently
// The first error encountered is returned
func Call[T any, U any](mc *MulticallClient, callOpts *bind.CallOpts, method abi.Method, target common.Address, inputs []T, parseFunc func([]any) U, maxBatchSize int) ([]U, error) {

	// apply defaults
	multicallAddress := mc.MulticallAddress
	maxConcurrentCalls := mc.MaxConcurrentCalls
	if multicallAddress == zeroAddress {
		multicallAddress = DefaultMulticallAddr
	}
	if maxConcurrentCalls == 0 {
		maxConcurrentCalls = DefaultMaxConcurrentCalls
	}

	// Batch inputs to multicall contract
	// split into smaller queries if over maximum batch and fetch concurrently
	var wg errgroup.Group
	wg.SetLimit(maxConcurrentCalls) // sensible concurrency limit
	outputs := make([]U, len(inputs))
	for batchStartIdx := 0; batchStartIdx < len(inputs); batchStartIdx += maxBatchSize {
		batchStart := batchStartIdx
		batchEnd := min(batchStartIdx+maxBatchSize, len(inputs))
		wg.Go(func(batchStart int, batchEnd int) func() error {
			return func() error {
				// execute batched multicall
				subOutputs, err := executeSubcall(mc.Client, multicallAddress, callOpts, method, target, inputs[batchStart:batchEnd], parseFunc)
				if err != nil {
					return fmt.Errorf("multicall subcall: %w", err)
				}

				// this is thread safe because each goroutine writes to a non-overlapping section of the slice
				copy(outputs[batchStart:], subOutputs)
				return nil
			}
		}(batchStart, batchEnd))
	}

	return outputs, wg.Wait()
}

func executeSubcall[T any, U any](client *ethclient.Client, multicallAddress common.Address, callOpts *bind.CallOpts, method abi.Method, target common.Address, inputs []T, parseFunc func([]any) U) ([]U, error) {
	multicallContract, err := NewMulticallCaller(multicallAddress, client)
	if err != nil {
		return nil, fmt.Errorf("binding multicall contract: %v", err)
	}

	// build batch of calls
	var calls []Multicall3Call
	for _, in := range inputs {

		var packedArgs []byte
		var err error

		// Gross but if T is a slice type I need to split it into its component parts
		if reflect.TypeOf(in).Kind() == reflect.Slice {
			s := reflect.ValueOf(in)

			// Pack each slice input individually
			ifaceSlice := make([]any, s.Len())
			for i := range ifaceSlice {
				ifaceSlice[i] = s.Index(i).Interface()
			}
			packedArgs, err = method.Inputs.Pack(ifaceSlice...)
		} else {
			// If T is not a slice, pack as normal
			packedArgs, err = method.Inputs.Pack(in)
		}
		if err != nil {
			return nil, fmt.Errorf("packing arguments for multicall: %v", err)
		}

		data := append(method.ID, packedArgs...)
		calls = append(calls, Multicall3Call{
			Target:   target,
			CallData: data,
		})
	}

	// call multicall contract with bundle
	var results []any
	err = multicallContract.Contract.Call(callOpts, &results, "aggregate", calls)
	if err != nil {
		return nil, fmt.Errorf("calling aggregate with queries: %v", err)
	}

	// use the provided parseFunc to interpret result data
	subResults := results[1].([][]byte) // aggregate() returns (blocknum, []results)
	var output []U
	for _, subResult := range subResults {
		fields, err := method.Outputs.Unpack(subResult)
		if err != nil {
			// defer to parse function on whether to blow up on empty input
			fmt.Printf("multicall subresult returned no data at block: %s\n", callOpts.BlockNumber)
			fields = []any{}
		}

		output = append(output, parseFunc(fields))
	}

	return output, nil
}
