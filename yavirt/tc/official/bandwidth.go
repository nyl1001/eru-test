package official

import (
	"encoding/json"
	"eru-test/yavirt/tc/official/log"
	"net"
	"strings"

	"github.com/florianl/go-tc"
	"golang.org/x/sys/unix"
)

type Bandwidth struct {
	PublicBandwidthAvg  uint32
	PublicBandwidthCeil uint32
}

func GenTcBandwidthConfig(ifaceName string, bandwidthLimitInfo *Bandwidth) error {
	devID, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Errorf("[GenTcBandwidthConfig] could not get interface ID, err: %v", err)
		return err
	}
	// open a rtnetlink socket
	tcSocket, err := tc.Open(&tc.Config{})
	if err != nil {
		log.Errorf("[GenTcBandwidthConfig] could not open rtnetlink socket, err: %v", err)
		return err
	}
	defer func() {
		if err := tcSocket.Close(); err != nil {
			log.Errorf("[GenTcBandwidthConfig] could not close rtnetlink socket, err: %v", err)
		}
	}()

	parentHtbQDISC, err := findRootHtbQDisc(tcSocket, ifaceName)
	if err != nil {
		return err
	}

	err = removeU32Filters(tcSocket, devID, parentHtbQDISC)
	if err != nil {
		return err
	}

	err = removeSfqQDiscs(tcSocket, ifaceName)
	if err != nil {
		return err
	}

	err = removeClasses(tcSocket, devID, parentHtbQDISC)
	if err != nil {
		return err
	}

	rootCls, err := findDefaultRootClass(tcSocket, devID, parentHtbQDISC)
	if err != nil {
		return err
	}

	classes, err := addClasses(tcSocket, rootCls, devID, bandwidthLimitInfo)
	if err != nil {
		return err
	}

	err = addSfqQDisc(tcSocket, devID, classes)
	if err != nil {
		return err
	}

	err = addFilters(tcSocket, devID, classes)
	if err != nil {
		return err
	}

	return nil
}

func RemoveTcBandwidthConfig(ifaceName string) error {
	devID, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Errorf("[RemoveTcBandwidthConfig] could not get interface ID, err: %v", err)
		return err
	}
	// open a rtnetlink socket
	tcSocket, err := tc.Open(&tc.Config{})
	if err != nil {
		log.Errorf("[RemoveTcBandwidthConfig] could not open rtnetlink socket, err: %v", err)
		return err
	}
	defer func() {
		if err := tcSocket.Close(); err != nil {
			log.Errorf("[RemoveTcBandwidthConfig] could not close rtnetlink socket, err: %v", err)
		}
	}()

	parentHtbQDISC, err := findRootHtbQDisc(tcSocket, ifaceName)
	if err != nil {
		return err
	}

	err = removeU32Filters(tcSocket, devID, parentHtbQDISC)
	if err != nil {
		return err
	}

	err = removeSfqQDiscs(tcSocket, ifaceName)
	if err != nil {
		return err
	}

	err = removeClasses(tcSocket, devID, parentHtbQDISC)
	if err != nil {
		return err
	}

	return nil
}

func findRootHtbQDisc(tcSocket *tc.Tc, ifaceName string) (*tc.Object, error) {
	qdiscs, err := tcSocket.Qdisc().Get()
	if err != nil {
		log.Errorf("[findRootHtbQDisc] could not get qdisc, err: %v", err)
		return nil, err
	}
	var distHtbQDISC tc.Object
	for _, qdisc := range qdiscs {
		iface, err := net.InterfaceByIndex(int(qdisc.Ifindex))
		if err != nil {
			log.Errorf("[findRootHtbQDisc] could not get interface from id %d, err: %v", qdisc.Ifindex, err)
			return nil, err
		}
		if iface.Name != ifaceName {
			continue
		}
		if iface.Name == ifaceName && qdisc.Kind == "htb" && qdisc.Parent == tc.HandleRoot {
			distHtbQDISC = qdisc
			break
		}
	}

	if distHtbQDISC.Handle == 0 {
		distHtbQDISC.Parent = tc.HandleRoot
	}
	return &distHtbQDISC, nil
}

func removeSfqQDiscs(tcSocket *tc.Tc, ifaceName string) error {
	qdiscs, err := tcSocket.Qdisc().Get()
	if err != nil {
		log.Errorf("[removeSfqQDiscs] could not get qdisc, err: %v", err)
		return err
	}
	for _, qdisc := range qdiscs {
		iface, err := net.InterfaceByIndex(int(qdisc.Ifindex))
		if err != nil {
			log.Errorf("[removeSfqQDiscs] could not get interface from id %d, err: %v", qdisc.Ifindex, err)
			return err
		}
		if iface.Name != ifaceName {
			continue
		}
		if qdisc.Parent == tc.HandleRoot {
			continue
		}
		err = tcSocket.Qdisc().Delete(&qdisc)
		if err != nil {
			log.Errorf("[removeSfqQDiscs] could not del qdisc, err: %v", err)
			return err
		}
	}
	return nil
}

func addClasses(tcSocket *tc.Tc, rootCls *tc.Object, devID *net.Interface, bandwidthLimitInfo *Bandwidth) ([]tc.Object, error) {
	publicIpRate := bandwidthLimitInfo.PublicBandwidthAvg
	publicIpCeil := bandwidthLimitInfo.PublicBandwidthCeil

	// It must be the class handle index in the specified range. To avoid duplication, specify it directly.
	constClsHandleIndex := rootCls.Handle + 1
	classes := []tc.Object{
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				Handle:  constClsHandleIndex + 1,
				Parent:  rootCls.Handle,
			},
			Attribute: tc.Attribute{
				Kind: "htb",
				Htb: &tc.Htb{
					Parms: &tc.HtbOpt{
						Rate: tc.RateSpec{
							CellLog:   0,
							Linklayer: 1,
							Overhead:  0,
							CellAlign: 0,
							Mpu:       0,
							Rate:      publicIpRate,
						},
						Ceil: tc.RateSpec{
							CellLog:   0,
							Linklayer: 1,
							Overhead:  0,
							CellAlign: 0,
							Mpu:       0,
							Rate:      publicIpCeil,
						},
						Buffer:  96000,
						Cbuffer: 20000,
						Quantum: 200000,
					},
				},
			},
		},
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				Handle:  constClsHandleIndex + 2,
				Parent:  rootCls.Handle,
			},
			Attribute: rootCls.Attribute,
		},
	}

	for _, cls := range classes {
		if err := tcSocket.Class().Add(&cls); err != nil {
			log.Errorf("[GenTcBandwidthConfig] add class failed, kind: %20s\thandle:%d\tparent:%d, err: %v", cls.Kind, cls.Handle, cls.Parent, err)
			return nil, err
		}
	}
	return classes, nil
}

func addSfqQDisc(tcSocket *tc.Tc, devID *net.Interface, classes []tc.Object) error {
	qdiscs := []tc.Object{
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				Parent:  classes[0].Handle,
				Info:    0,
			},
			Attribute: tc.Attribute{
				Kind: "sfq", Sfq: &tc.Sfq{V0: tc.SfqQopt{
					PerturbPeriod: 10,
					Limit:         3000,
					Flows:         512,
				},
				},
			},
		},
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				Parent:  classes[1].Handle,
				Info:    0,
			},
			Attribute: tc.Attribute{
				Kind: "sfq", Sfq: &tc.Sfq{V0: tc.SfqQopt{
					PerturbPeriod: 10,
					Limit:         3000,
					Flows:         512,
				},
				},
			},
		},
	}

	for _, qs := range qdiscs {
		if err := tcSocket.Qdisc().Add(&qs); err != nil {
			log.Errorf("[addClasses] add qdisc failed, kind: %20s\thandle:%d\tparent:%d, err: %v", qs.Kind, qs.Handle, qs.Parent, err)
			return err
		}
	}
	return nil
}

func removeU32Filters(tcSocket *tc.Tc, devID *net.Interface, parentHtbQDISC *tc.Object) error {
	relatedClasses, err := tcSocket.Class().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
		Handle:  0,
		Parent:  parentHtbQDISC.Handle,
		Info:    0,
	})
	if err != nil {
		return err
	}
	filters, _ := tcSocket.Filter().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
	})
	for _, ft := range filters {
		err := tcSocket.Filter().Delete(&ft)
		if err != nil {
			if strings.Contains(err.Error(), "no such file or directory") {
				err = nil
				continue
			}
			by, _ := json.Marshal(ft)
			log.Errorf("[removeU32Filters] delete top filter failed, err: kind: %v, %20s handle:%d parent:%d, top filter info : %s", err, ft.Kind, ft.Handle, ft.Parent, string(by))
			return err
		}
	}
	for _, pCls := range relatedClasses {
		filters, _ := tcSocket.Filter().Get(&tc.Msg{
			Family:  0,
			Ifindex: uint32(devID.Index),
			Parent:  pCls.Handle,
		})
		for _, ft := range filters {
			err := tcSocket.Filter().Delete(&ft)
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					err = nil
					continue
				}
				by, _ := json.Marshal(ft)
				log.Errorf("[removeU32Filters] delete child filter failed, err: kind: %v, %20s handle:%d parent:%d, top filter info : %s", err, ft.Kind, ft.Handle, ft.Parent, string(by))
				return err
			}
		}
	}
	return nil
}

func findDefaultRootClass(tcSocket *tc.Tc, devID *net.Interface, parentHtbQDISC *tc.Object) (*tc.Object, error) {
	relatedClasses, err := tcSocket.Class().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
		Handle:  0,
		Parent:  parentHtbQDISC.Handle,
		Info:    0,
	})
	if err != nil {
		return nil, err
	}
	var rootCls tc.Object
	for _, cls := range relatedClasses {
		if cls.Parent == tc.HandleRoot {
			rootCls = cls
			continue
		}
		err = tcSocket.Class().Delete(&cls)
		if err != nil {
			log.Errorf("[findDefaultRootClass] could not del class, err: %v", err)
			return nil, err
		}
	}
	return &rootCls, nil
}

func removeClasses(tcSocket *tc.Tc, devID *net.Interface, parentHtbQDISC *tc.Object) error {
	relatedClasses, err := tcSocket.Class().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
		Handle:  0,
		Parent:  parentHtbQDISC.Handle,
		Info:    0,
	})
	if err != nil {
		return err
	}
	for _, cls := range relatedClasses {
		if cls.Parent == tc.HandleRoot {
			continue
		}
		err = tcSocket.Class().Delete(&cls)
		if err != nil {
			log.Errorf("[removeClasses] could not del class, err: %v", err)
			return err
		}
	}
	return nil
}

func addFilters(tcSocket *tc.Tc, devID *net.Interface, classes []tc.Object) error {
	infos := []uint32{131080, 65544} // 对应 ip protocol pref 的两个数值 2，1
	u32Filters := []tc.Object{
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				Handle:  0,
				Info:    infos[0],
			},
			Attribute: tc.Attribute{
				Kind: "u32",
				U32: &tc.U32{
					ClassID: &classes[0].Handle,
					// 匹配在0.0.0.0/0范围内的IP地址
					Sel: &tc.U32Sel{
						Flags:   0x1,
						NKeys:   2,
						Off:     0, // 偏移量，跳过以太网头和IP头
						OffMask: 0, // 不使用子网掩码
						Offoff:  0,
						Keys: []tc.U32Key{
							{
								Mask:    0x00000000, // 不使用子网掩码
								Val:     0x00000000, // 0.0.0.0的二进制表示
								Off:     16,         // 表示目的ip，dist ip
								OffMask: 0,
							},
							{
								Mask:    0x00000000, // 不使用子网掩码
								Val:     0x00000000, // 0.0.0.0的二进制表示
								Off:     12,         // 表示来源ip，src ip
								OffMask: 0,
							},
						},
					},
				},
				Prio: &tc.Prio{Bands: 0}, // 设置优先级，例如2个band
			},
		},
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				Handle:  0,
				Info:    infos[1],
			},
			Attribute: tc.Attribute{
				Kind: "u32",
				U32: &tc.U32{
					ClassID: &classes[1].Handle,
					// 匹配在10.0.0.0/8范围内的IP地址
					Sel: &tc.U32Sel{
						Flags:   1,
						NKeys:   2,
						Off:     0, // 偏移量，跳过以太网头和IP头
						OffMask: 0, // 8位子网掩码
						Offoff:  0,
						Keys: []tc.U32Key{
							{
								Mask:    0x000000ff, // 8位子网掩码
								Val:     0x0000000a, // 10.0.0.0的二进制表示
								Off:     16,         // 表示目的ip，dist ip
								OffMask: 0,
							},
							{
								Mask:    0x000000ff, // 8位子网掩码
								Val:     0x0000000a, // 10.0.0.0的二进制表示
								Off:     12,         // 表示来源ip，src ip
								OffMask: 0,
							},
						},
					},
				},
				Prio: &tc.Prio{Bands: 0}, // 设置优先级，例如1个band
			},
		},
	}

	for _, uf := range u32Filters {
		if err := tcSocket.Filter().Add(&uf); err != nil {
			log.Errorf("[addFilters] add u32 filter failed, kind: %20s\thandle:%d\tparent:%d, err: %v", uf.Kind, uf.Handle, uf.Parent, err)
			return err
		}
	}
	return nil
}
