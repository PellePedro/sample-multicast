package flow

import (
	hal "github.com/drivenets/vmw_tsf/tsf-hal"
)

type FLowHandler struct {
	client hal.DnHal
}

func NewFlowHandler() *FLowHandler {
	return &FLowHandler{
		client: hal.NewDnHal(),
	}
}

func (fh *FLowHandler) CreateHalClient() {
	fh.client = hal.NewDnHal()
}
func (fh FLowHandler) Steer(key *hal.FlowKey, nextHop string) {
	fh.client.Steer(key, nextHop)
}

func (fh FLowHandler) getFlows(flowCh chan interface{}) {
	fh.client.GetFlows(func(key *hal.FlowKey, stat *hal.FlowTelemetry) error {
		f := FlowTelemetry{
			SrcAddr:      key.SrcAddr,
			DstAddr:      key.DstAddr,
			DstPort:      key.DstPort,
			Protocol:     key.Protocol,
			IngressIf:    stat.IngressIf,
			EgressIf:     stat.EgressIf,
			RxRatePps:    stat.RxRatePps,
			RxTotalPkts:  stat.RxTotalPkts,
			RxRateBps:    stat.RxRateBps,
			RxTotalBytes: stat.RxTotalBytes,
		}
		flowCh <- f

		return nil
	})
}

func (fh FLowHandler) GetInterfaces(flowCh chan interface{}) {
	fh.client.GetInterfaces(
		func(ifc string, tm *hal.InterfaceTelemetry) error {
			t := InterfaceTelemetry{
				Speed:   tm.Speed,
				RxBytes: tm.RxBytes,
				RxBps:   tm.RxBps,
				TxBps:   tm.RxBps,
				TxBytes: tm.TxBytes,
				Link: LinkTelemetry{
					Delay:  tm.Link.Delay,
					Jitter: tm.Link.Jitter,
				},
			}
			flowCh <- t
			return nil
		})
}
