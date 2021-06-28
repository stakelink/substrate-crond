package cron

import (
	"fmt"
	"math"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
)

type SubstrateUtils struct {
	meta *types.Metadata
	gsrpc.SubstrateAPI
}

type SessionInfo struct{
	Config struct {
		GenesisSlot uint64
		Duration uint64
		SessionsPerEra uint64
	}

	CurrentSlot uint64
	CurrentIndex uint64
	CurrentEra uint64
	CurrentStart uint64
}

func NewSubstrateUtils(url string) (*SubstrateUtils, error) {
	api, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	return &SubstrateUtils{
		meta,
		gsrpc.SubstrateAPI(*api),
	}, nil
}

func (api *SubstrateUtils) Reconnect() (error) {
	newapi, err := NewSubstrateUtils(api.Client.URL())
	if err != nil {
		fmt.Println(err)
		return err
	}
	api = newapi
	return nil
}


func (api *SubstrateUtils) GetConstant(module string, constant string, dest interface{}) (error) {
	switch api.meta.Version {
		case 12:
			return getConstantV12(api.meta, module, constant, dest)
		case 13:
			return getConstantV13(api.meta, module, constant, dest)
	}
	return fmt.Errorf("metadata version not suported [%d]", api.meta.Version)
}

func getConstantV12(meta *types.Metadata, module string, constant string, dest interface{}) (error) {
	for _,m := range meta.AsMetadataV12.Modules {
		if string(m.Name) == module {
			for _,c := range m.Constants {
				if string(c.Name) == constant {
					return types.DecodeFromBytes(c.Value, dest)
				}
			}
		}
	}
	return fmt.Errorf("constant not available")
}

func getConstantV13(meta *types.Metadata, module string, constant string, dest interface{}) (error) {
	for _,m := range meta.AsMetadataV13.Modules {
		if string(m.Name) == module {
			for _,c := range m.Constants {
				if string(c.Name) == constant {
					return types.DecodeFromBytes(c.Value, dest)
				}
			}
		}
	}
	return fmt.Errorf("constant not available")
}


func (api *SubstrateUtils) GetStorage(module string, method string,  arg1 []byte, arg2 []byte, dest interface{}) (error) {
	key, err := types.CreateStorageKey(api.meta, module, method, arg1, arg2)
	if err != nil {
		return err
	}

	_, err = api.RPC.State.GetStorageLatest(key, dest)
	if err != nil {
		return err
	}

	return nil
}


func (api *SubstrateUtils) GetSessionInfo() (*SessionInfo, error) {
	var GenesisSlot types.U64
	err := api.GetStorage("Babe", "GenesisSlot", nil, nil, &GenesisSlot)
	if err != nil {
		return nil, err
	}

	var EpochDuration types.U64
	err = api.GetConstant("Babe", "EpochDuration", &EpochDuration)
	if err != nil {
		return nil, err
	}

	var SessionsIndex types.U32
	err = api.GetConstant("Staking", "SessionsPerEra", &SessionsIndex)
	if err != nil {
		return nil, err
	}

	var CurrentSlot types.U64
	err = api.GetStorage("Babe", "CurrentSlot", nil, nil, &CurrentSlot)
	if err != nil {
		return nil, err
	}

	var CurrentIndex types.U32
	err = api.GetStorage("Session", "CurrentIndex", nil, nil, &CurrentIndex)
	if err != nil {
		return nil, err
	}


	var CurrentEra types.U32
	err = api.GetStorage("Staking", "CurrentEra", nil, nil, &CurrentEra)
	if err != nil {
		return nil, err
	}

	var EpochStart [2]BlockNumber
	err = api.GetStorage("Babe", "EpochStart", nil, nil, &EpochStart)
	if err != nil {
		panic(err)
	}

	info := &SessionInfo{
		CurrentSlot: uint64(CurrentSlot),
		CurrentIndex: uint64(CurrentIndex),
		CurrentEra: uint64(CurrentEra),
		CurrentStart: uint64(EpochStart[1]),
	}
	info.Config.GenesisSlot = uint64(GenesisSlot)
	info.Config.Duration = uint64(EpochDuration)
	info.Config.SessionsPerEra = uint64(SessionsIndex)

	return info, nil
}

func (info *SessionInfo) GetLocalSlot() uint64 {
	return (info.CurrentSlot - info.Config.GenesisSlot - info.CurrentIndex * info.Config.Duration)
}

func (info *SessionInfo) GetLocalSession() uint64 {
	return uint64(math.Mod(float64(info.CurrentIndex), float64(info.Config.SessionsPerEra)))
}

func (info *SessionInfo) GetCurrentEra() uint64 {
	return uint64(info.CurrentEra)
}