package metrics

import (
	"errors"
	"net"
	"net/http"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/experimental"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _ adapter.MetricService = (*metricServer)(nil)

func init() {
	experimental.RegisterMetricServerConstructor(NewServer)
}

type metricServer struct {
	http *http.Server

	logger log.Logger
	opts   option.MetricOptions

	registry *prometheus.Registry

	packetCountersInbound  *prometheus.CounterVec
	packetCountersOutbound *prometheus.CounterVec
}

func NewServer(logger log.Logger, opts option.MetricOptions) (adapter.MetricService, error) {
	r := chi.NewRouter()
	_server := &http.Server{
		Addr:    opts.Listen,
		Handler: r,
	}
	if opts.Path == "" {
		opts.Path = "/metrics"
	}

	// Create a custom registry
	registry := prometheus.NewRegistry()

	// Add Go metrics to the registry
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	r.Get(opts.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP)
	server := &metricServer{
		http:     _server,
		logger:   logger,
		opts:     opts,
		registry: registry,
	}
	err := server.registerMetrics()
	return server, err
}

func (s *metricServer) Name() string {
	return "metric-api"
}

func (s *metricServer) Start(stage adapter.StartStage) error {
	if !s.opts.Enabled() {
		return nil
	}
	go func() {
		err := s.http.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			if errors.Is(err, net.ErrClosed) {
				s.logger.Debug("metrics api server closed")
			} else {
				s.logger.Error("metrics api listen error", err)
			}
		} else {
			s.logger.Info("metrics api listening at ", s.http.Addr, s.opts.Path)
		}
	}()
	return nil
}

func (s *metricServer) Close() error {
	if !s.opts.Enabled() {
		return nil
	}
	return common.Close(common.PtrOrNil(s.http))
}
