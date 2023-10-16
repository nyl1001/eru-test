package yavirt

import (
	"context"
	"encoding/json"
	"eru-test/core"

	corepb "github.com/projecteru2/core/rpc/gen"
)

func GetClient(ctx context.Context) (corepb.CoreRPCClient, error) {
	err := core.Prepare(ctx)
	if err != nil {
		return nil, err
	}
	return core.Get().GetClient(), nil
}

func RawEngine(ctx context.Context, wlID, opStr string, extendParams any) (*corepb.RawEngineMessage, error) {
	extendParamsBytes, err := json.Marshal(extendParams)
	if err != nil {
		return nil, err
	}
	opts := corepb.RawEngineOptions{
		Id:     wlID,
		Op:     opStr,
		Params: extendParamsBytes,
	}
	eruClient, err := GetClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := eruClient.RawEngine(ctx, &opts)
	if err != nil {
		return nil, err
	}
	return resp, err
}
