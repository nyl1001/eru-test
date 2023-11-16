package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

func main() {
	//showIfTcConfigTest()
	//return
	ifaceName := "calif0779ce47ed"
	err := showTcBandwidthConfig(ifaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "showTcBandwidthConfig error: %v\n", err)
		return
	}
	err = genTcBandwidthConfig(ifaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genTcBandwidthConfig error: %v\n", err)
		return
	}
}

func showIfTcConfigTest() {
	fmt.Println("core.BuildHandle(0x1000, 0x00)", core.BuildHandle(0xF, 0x0))
	ifaceName := "cali5a8d02067ec"
	devID, err := net.InterfaceByName(ifaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get interface ID: %v\n", err)
		return
	}

	fmt.Println(devID)
	// open a rtnetlink socket
	tcSocket, err := tc.Open(&tc.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open rtnetlink socket: %v\n", err)
		return
	}
	defer func() {
		if err := tcSocket.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "could not close rtnetlink socket: %v\n", err)
		}
	}()

	err = tcSocket.SetOption(netlink.ExtendedAcknowledge, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not set option ExtendedAcknowledge: %v\n", err)
		return
	}

	qdiscs, err := tcSocket.Qdisc().Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get qdiscs: %v\n", err)
		return
	}
	var distQDISC tc.Object
	for _, qdisc := range qdiscs {
		iface, err := net.InterfaceByIndex(int(qdisc.Ifindex))
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get interface from id %d: %v", qdisc.Ifindex, err)
			return
		}
		if !strings.HasPrefix(iface.Name, "cali") {
			continue
		}
		if iface.Name != ifaceName {
			continue
		}
		fmt.Printf("qdsic => %20s\t%s\thandle:%d\tparent:%d\n", iface.Name, qdisc.Kind, qdisc.Handle, qdisc.Parent)
		//qdiscBy, _ := json.Marshal(qdisc)
		//fmt.Println("qdiscBy:", string(qdiscBy))
		if iface.Name == ifaceName && qdisc.Kind == "htb" {
			distQDISC = qdisc
		}
	}

	classes, err := tcSocket.Class().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
		Handle:  distQDISC.Parent,
		Parent:  0,
		Info:    0,
	})

	for _, cls := range classes {
		fmt.Printf("class => %20s\thandle:%d\tparent:%d\n", cls.Kind, cls.Handle, cls.Parent)
		ftBytes, _ := json.Marshal(cls)
		fmt.Printf("cls details: %+v\n", string(ftBytes))
	}

	filters, err := tcSocket.Filter().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
		Handle:  distQDISC.Parent,
		Parent:  0,
		Info:    0,
	})

	for _, ft := range filters {
		fmt.Printf("filter => %20s\thandle:%d\tparent:%d\n", ft.Kind, ft.Handle, ft.Parent)
		ftBytes, _ := json.Marshal(ft)
		fmt.Printf("filter details: %+v\n", string(ftBytes))
	}

}

func showTcBandwidthConfig(ifaceName string) error {
	devID, err := net.InterfaceByName(ifaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get interface ID: %v\n", err)
		return err
	}

	fmt.Println(devID)
	// open a rtnetlink socket
	tcSocket, err := tc.Open(&tc.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open rtnetlink socket: %v\n", err)
		return err
	}
	defer func() {
		if err := tcSocket.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "could not close rtnetlink socket: %v\n", err)
		}
	}()

	qdiscs, err := tcSocket.Qdisc().Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get qdiscs: %v\n", err)
		return err
	}
	var distHtbQDISC tc.Object
	for _, qdisc := range qdiscs {
		iface, err := net.InterfaceByIndex(int(qdisc.Ifindex))
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get interface from id %d: %v", qdisc.Ifindex, err)
			return err
		}
		if iface.Name != ifaceName {
			continue
		}
		fmt.Printf("qdsic => %20s\t%s\thandle:%d\tparent:%d\n", iface.Name, qdisc.Kind, qdisc.Handle, qdisc.Parent)
		if iface.Name == ifaceName && qdisc.Kind == "htb" && qdisc.Parent == tc.HandleRoot {
			distHtbQDISC = qdisc
		}
	}

	if distHtbQDISC.Handle == 0 {
		distHtbQDISC.Parent = tc.HandleRoot
	}

	relatedClasses, err := tcSocket.Class().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
		Handle:  0,
		Parent:  distHtbQDISC.Handle,
		Info:    0,
	})

	err = displayFilters(tcSocket, devID, relatedClasses)
	if err != nil {
		return err
	}

	return nil
}

func genTcBandwidthConfig(ifaceName string) error {
	devID, err := net.InterfaceByName(ifaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get interface ID: %v\n", err)
		return err
	}

	fmt.Println(devID)
	// open a rtnetlink socket
	tcSocket, err := tc.Open(&tc.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open rtnetlink socket: %v\n", err)
		return err
	}
	defer func() {
		if err := tcSocket.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "could not close rtnetlink socket: %v\n", err)
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

	bandwidthLimitInfo := Bandwidth{
		TotalBandwidthAvg:   uint32(20971520 / 8),
		TotalBandwidthCeil:  uint32(10485760 / 8),
		PublicBandwidthAvg:  uint32(5971520 / 8),
		PublicBandwidthCeil: uint32(10971520 / 8),
	}

	classes, err := addClasses(tcSocket, rootCls, devID, &bandwidthLimitInfo)
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

func findRootHtbQDisc(tcSocket *tc.Tc, ifaceName string) (*tc.Object, error) {
	qdiscs, err := tcSocket.Qdisc().Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get qdiscs: %v\n", err)
		return nil, err
	}
	var distHtbQDISC tc.Object
	for _, qdisc := range qdiscs {
		iface, err := net.InterfaceByIndex(int(qdisc.Ifindex))
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get interface from id %d: %v", qdisc.Ifindex, err)
			return nil, err
		}
		if iface.Name != ifaceName {
			continue
		}
		fmt.Printf("qdsic => %20s\t%s\thandle:%d\tparent:%d\n", iface.Name, qdisc.Kind, qdisc.Handle, qdisc.Parent)
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
		fmt.Fprintf(os.Stderr, "could not get qdiscs: %v\n", err)
		return err
	}
	for _, qdisc := range qdiscs {
		iface, err := net.InterfaceByIndex(int(qdisc.Ifindex))
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get interface from id %d: %v", qdisc.Ifindex, err)
			return err
		}
		if iface.Name != ifaceName {
			continue
		}
		if qdisc.Parent == tc.HandleRoot {
			fmt.Printf("child qdisc [reserve] => %20s\thandle:%d\tparent:%d\n", qdisc.Kind, qdisc.Handle, qdisc.Parent)
			continue
		}
		fmt.Printf("child qdisc [delete] => %20s\thandle:%d\tparent:%d\n", qdisc.Kind, qdisc.Handle, qdisc.Parent)
		err = tcSocket.Qdisc().Delete(&qdisc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not del qdisc: %v\n", err)
			return err
		}
	}
	return nil
}

type Bandwidth struct {
	TotalBandwidthAvg   uint32
	TotalBandwidthCeil  uint32
	PublicBandwidthAvg  uint32
	PublicBandwidthCeil uint32
}

func addClasses(tcSocket *tc.Tc, rootCls *tc.Object, devID *net.Interface, bandwidthLimitInfo *Bandwidth) ([]tc.Object, error) {
	publicIpRate := bandwidthLimitInfo.PublicBandwidthAvg
	publicIpCeil := bandwidthLimitInfo.PublicBandwidthCeil

	totalBandwidthAvg := bandwidthLimitInfo.TotalBandwidthAvg
	totalBandwidthCeil := bandwidthLimitInfo.TotalBandwidthCeil

	constClsHandleIndex := uint32(65536)
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
				Stats: &tc.Stats{
					Bytes:      48622036413,
					Packets:    104418,
					Drops:      4,
					Overlimits: 750796,
					Bps:        0,
					Pps:        0,
					Qlen:       0,
					Backlog:    0,
				},
				XStats: &tc.XStats{
					Htb: &tc.HtbXStats{
						Lends:   0,
						Borrows: 0,
						Giants:  0,
						Tokens:  625,
						CTokens: 62,
					},
				},
				Stats2: &tc.Stats2{
					Bytes:      65556,
					Packets:    0,
					Qlen:       0,
					Backlog:    0,
					Drops:      196632,
					Requeues:   0,
					Overlimits: 0,
				},
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
			Attribute: tc.Attribute{
				Kind: "htb",
				Stats: &tc.Stats{
					Bytes:      48622036413,
					Packets:    104418,
					Drops:      4,
					Overlimits: 750796,
					Bps:        0,
					Pps:        0,
					Qlen:       0,
					Backlog:    0,
				},
				XStats: &tc.XStats{
					Htb: &tc.HtbXStats{
						Lends:   0,
						Borrows: 0,
						Giants:  0,
						Tokens:  625,
						CTokens: 62,
					},
				},
				Stats2: &tc.Stats2{
					Bytes:      65556,
					Packets:    0,
					Qlen:       0,
					Backlog:    0,
					Drops:      196632,
					Requeues:   0,
					Overlimits: 0,
				},
				Htb: &tc.Htb{
					Parms: &tc.HtbOpt{
						Rate: tc.RateSpec{
							CellLog:   0,
							Linklayer: 1,
							Overhead:  0,
							CellAlign: 0,
							Mpu:       0,
							Rate:      totalBandwidthAvg,
						},
						Ceil: tc.RateSpec{
							CellLog:   0,
							Linklayer: 1,
							Overhead:  0,
							CellAlign: 0,
							Mpu:       0,
							Rate:      totalBandwidthCeil,
						},
						Buffer:  96000,
						Cbuffer: 20000,
						Quantum: 200000,
					},
				},
			},
		},
	}

	for _, cls := range classes {
		//clsb, _ := json.Marshal(cls)
		//fmt.Printf("Trying to add class: %s\n", string(clsb))
		if err := tcSocket.Class().Add(&cls); err != nil {
			fmt.Fprintf(os.Stderr, "add class failed, kind: %20s\thandle:%d\tparent:%d, error: %v\n", cls.Kind, cls.Handle, cls.Parent, err)
			return nil, err
		}
		fmt.Printf("add class success => kind: %20s\thandle:%d\tparent:%d\n", cls.Kind, cls.Handle, cls.Parent)

	}
	return classes, nil
}

func addSfqQDisc(tcSocket *tc.Tc, devID *net.Interface, classes []tc.Object) error {
	qdiscs := []tc.Object{
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				//Handle:  uint32(1048576),
				Parent: classes[0].Handle,
				Info:   0,
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
				//Handle:  uint32(2097152),
				Parent: classes[1].Handle,
				Info:   0,
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
		//qsb, _ := json.Marshal(qs)
		//fmt.Printf("Trying to add qdisc: %s\n", string(qsb))
		if err := tcSocket.Qdisc().Add(&qs); err != nil {
			fmt.Fprintf(os.Stderr, "add qdisc failed, kind: %20s\thandle:%d\tparent:%d, error: %v\n", qs.Kind, qs.Handle, qs.Parent, err)
			return err
		}
		fmt.Printf("add qdisc success => kind: %20s\thandle:%d\tparent:%d\n", qs.Kind, qs.Handle, qs.Parent)
	}
	return nil
}

func displayFilters(tcSocket *tc.Tc, devID *net.Interface, parentClasses []tc.Object) error {
	filters, _ := tcSocket.Filter().Get(&tc.Msg{
		Family:  0,
		Ifindex: uint32(devID.Index),
	})
	for _, ft := range filters {
		fmt.Printf(" filter => %20s\thandle:%d\tparent:%d\n", ft.Kind, ft.Handle, ft.Parent)
	}
	for _, pCls := range parentClasses {
		filters, _ = tcSocket.Filter().Get(&tc.Msg{
			Family:  0,
			Ifindex: uint32(devID.Index),
			Parent:  pCls.Handle,
		})
		for _, ft := range filters {
			fmt.Printf(" filter => %20s\thandle:%d\tparent:%d\n", ft.Kind, ft.Handle, ft.Parent)
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
		if ft.Kind != "u32" {
			fmt.Printf("filter [reserve] => %20s\thandle:%d\tparent:%d\n", ft.Kind, ft.Handle, ft.Parent)
			continue
		}
		err := tcSocket.Filter().Delete(&ft)
		if err != nil {
			if strings.Contains(err.Error(), "no such file or directory") {
				err = nil
				continue
			}
			fmt.Fprintf(os.Stderr, "delete filter failed: %v, %20s\thandle:%d\tparent:%d \n", err, ft.Kind, ft.Handle, ft.Parent)
			return err
		}
		fmt.Printf("delete filter success => %20s\thandle:%d\tparent:%d\n", ft.Kind, ft.Handle, ft.Parent)
	}
	for _, pCls := range relatedClasses {
		filters, _ := tcSocket.Filter().Get(&tc.Msg{
			Family:  0,
			Ifindex: uint32(devID.Index),
			Parent:  pCls.Handle,
		})
		for _, ft := range filters {
			if ft.Kind != "u32" {
				fmt.Printf("filter [reserve] => %20s\thandle:%d\tparent:%d\n", ft.Kind, ft.Handle, ft.Parent)
				continue
			}
			err := tcSocket.Filter().Delete(&ft)
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					err = nil
					continue
				}
				fmt.Fprintf(os.Stderr, "delete filter failed: %v, %20s\thandle:%d\tparent:%d \n", err, ft.Kind, ft.Handle, ft.Parent)
				return err
			}
			fmt.Printf("delete filter success => %20s\thandle:%d\tparent:%d\n", ft.Kind, ft.Handle, ft.Parent)
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
			fmt.Printf("class [reserve] => %20s\thandle:%d\tparent:%d\n", cls.Kind, cls.Handle, cls.Parent)
			continue
		}
		fmt.Printf("class [delete] => %20s\thandle:%d\tparent:%d\n", cls.Kind, cls.Handle, cls.Parent)
		err = tcSocket.Class().Delete(&cls)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not del class: %v\n", err)
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
			fmt.Printf("class [reserve] => %20s\thandle:%d\tparent:%d\n", cls.Kind, cls.Handle, cls.Parent)
			continue
		}
		fmt.Printf("class [delete] => %20s\thandle:%d\tparent:%d\n", cls.Kind, cls.Handle, cls.Parent)
		err = tcSocket.Class().Delete(&cls)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not del class: %v\n", err)
			return err
		}
	}
	return nil
}

func addFilters(tcSocket *tc.Tc, devID *net.Interface, classes []tc.Object) error {
	u32Filters := []tc.Object{
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				//Handle:  uint32(2149582848),
				Handle: 0,
				//Parent: rootCls.Handle,
				Info: 3221094408,
			},
			Attribute: tc.Attribute{
				Kind: "u32",
				U32: &tc.U32{
					ClassID: &classes[0].Handle,
					// 匹配在0.0.0.0/0范围内的IP地址
					Sel: &tc.U32Sel{
						Flags:   0x1,
						NKeys:   1,
						Off:     0, // 偏移量，跳过以太网头和IP头
						OffMask: 0, // 不使用子网掩码
						Offoff:  0,
						Keys: []tc.U32Key{
							{
								Mask:    0x00000000, // 不使用子网掩码
								Val:     0x00000000, // 0.0.0.0的二进制表示
								Off:     16,
								OffMask: 0,
							},
						},
					},
				},
				Prio: &tc.Prio{Bands: 2}, // 设置优先级，例如2个band
			},
		},
		{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(devID.Index),
				//Handle:  uint32(2149580800),
				Handle: 0,
				//Parent: rootCls.Handle,
				Info: 3221094408,
			},
			Attribute: tc.Attribute{
				Kind: "u32",
				U32: &tc.U32{
					ClassID: &classes[1].Handle,
					// 匹配在10.0.0.0/8范围内的IP地址
					Sel: &tc.U32Sel{
						Flags:   1,
						NKeys:   1,
						Off:     0, // 偏移量，跳过以太网头和IP头
						OffMask: 0, // 8位子网掩码
						Offoff:  0,
						Keys: []tc.U32Key{
							{
								Mask:    0x000000ff, // 8位子网掩码
								Val:     0x0000000a, // 10.0.0.0的二进制表示
								Off:     12,
								OffMask: 0,
							},
						},
					},
				},
				Prio: &tc.Prio{Bands: 1}, // 设置优先级，例如1个band
			},
		},
	}

	for _, uf := range u32Filters {
		//ufb, _ := json.Marshal(uf)
		//fmt.Printf("Trying to add filter: %s\n", string(ufb))
		if err := tcSocket.Filter().Add(&uf); err != nil {
			fmt.Fprintf(os.Stderr, "add u32 filter failed, kind: %20s\thandle:%d\tparent:%d, error: %v\n", uf.Kind, uf.Handle, uf.Parent, err)
			return err
		}
		fmt.Printf("add u32 filter success => kind: %20s\thandle:%d\tparent:%d\n", uf.Kind, uf.Handle, uf.Parent)
	}
	return nil
}
