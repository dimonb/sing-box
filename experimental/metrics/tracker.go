package metrics

import (
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/bufio"
	N "github.com/sagernet/sing/common/network"

	"github.com/prometheus/client_golang/prometheus"
)

func (s *metricServer) registerMetrics() error {
	// Create a simple test counter first
	testCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "sing_box_test_metric",
		Help: "Test metric to verify registration works",
	})

	s.packetCountersInbound = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "inbound_packet_bytes",
		Help: "Total bytes of inbound packets",
	}, []string{"inbound", "user"})

	s.packetCountersOutbound = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "outbound_packet_bytes",
		Help: "Total bytes of outbound packets",
	}, []string{"outbound", "user"})

	var err error
	err = s.registry.Register(testCounter)
	if err != nil {
		return err
	}
	testCounter.Add(42)

	err = s.registry.Register(s.packetCountersInbound)
	if err != nil {
		return err
	}
	err = s.registry.Register(s.packetCountersOutbound)
	if err != nil {
		return err
	}

	// Initialize with zero values to make metrics visible
	s.packetCountersInbound.WithLabelValues("test", "").Add(0)
	s.packetCountersOutbound.WithLabelValues("test", "").Add(0)

	return nil
}

func (s *metricServer) WithConnCounters(inbound, outbound, user string) adapter.ConnAdapter[net.Conn] {
	incRead, incWrite := s.getPacketCounters(inbound, outbound, user)
	return func(conn net.Conn) net.Conn {
		return bufio.NewCounterConn(conn, []N.CountFunc{incRead}, []N.CountFunc{incWrite})
	}
}

func (s *metricServer) WithPacketConnCounters(inbound, outbound, user string) adapter.ConnAdapter[N.PacketConn] {
	incRead, incWrite := s.getPacketCounters(inbound, outbound, user)
	return func(conn N.PacketConn) N.PacketConn {
		return bufio.NewCounterPacketConn(conn, []N.CountFunc{incRead}, []N.CountFunc{incWrite})
	}
}

func (s *metricServer) getPacketCounters(inbound, outbound, user string) (
	readCounters N.CountFunc,
	writeCounters N.CountFunc,
) {
	return func(n int64) {
			s.packetCountersInbound.WithLabelValues(inbound, user).Add(float64(n))
		}, func(n int64) {
			s.packetCountersOutbound.WithLabelValues(outbound, user).Add(float64(n))
		}
}
