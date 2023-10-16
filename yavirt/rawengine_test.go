package yavirt

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

//func (svc *Boar) RawEngine(ctx context.Context, id string, req types.RawEngineReq) (types.RawEngineResp, error) {
//	switch req.Op {
//	case "vm-get-vnc-port":
//		return svc.getVNCPort(ctx, id)
//	case "vm-init-sys-disk":
//		return svc.InitSysDisk(ctx, id, req.Params)
//	case "vm-fs-freeze-all":
//		return svc.fsFreezeAll(ctx, id)
//	case "vm-fs-thaw-all":
//		return svc.fsThawAll(ctx, id)
//	case "vm-list-vols", "vm-list-vol", "vm-list-volume", "vm-list-volumes":
//		return svc.listVolumes(ctx, id)
//	case "vm-fs-freeze-status":
//		return svc.fsFreezeStatus(ctx, id)
//	default:
//		return types.RawEngineResp{}, errors.Errorf("invalid operation %s", req.Op)
//	}

func TestVmListVols(t *testing.T) {
	ctx := context.Background()
	eruWorkloadID := "00-ERU-YET-ANOTHER-VIRT-2023042100032016970114660352050000000001"
	commonRawEngineWithPrint(ctx, eruWorkloadID, "vm-list-vols", nil)
	commonRawEngineWithPrint(ctx, eruWorkloadID, "vm-fs-freeze-status", nil)
	commonRawEngineWithPrint(ctx, eruWorkloadID, "vm-fs-freeze-all", nil)
	commonRawEngineWithPrint(ctx, eruWorkloadID, "vm-fs-freeze-status", nil)
	commonRawEngineWithPrint(ctx, eruWorkloadID, "vm-fs-thaw-all", nil)
	commonRawEngineWithPrint(ctx, eruWorkloadID, "vm-fs-freeze-status", nil)
}

func commonRawEngineWithPrint(ctx context.Context, wlID, opStr string, extendParams any) {
	extendParamsBytes, _ := json.Marshal(extendParams)
	resp, err := RawEngine(ctx, wlID, opStr, nil)
	if err != nil {
		fmt.Println(fmt.Sprintf("【error】call RawEngine error, err msg: %s,  input params: ID : %s, op: %s, extendParams: %s", err.Error(), wlID, opStr, string(extendParamsBytes)))
		return
	}
	fmt.Println(fmt.Sprintf("【success】call RawEngine input params: ID : %s, op: %s, extendParams: %s, response: %v ", wlID, opStr, string(extendParamsBytes), resp))
}
