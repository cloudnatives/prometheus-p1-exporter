package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/skoef/gop1"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	listenAddress = kingpin.Flag(
		"listen-address",
		"Address on which to expose metrics and web interface.",
	).Default(":9832").String()
	deviceName = kingpin.Flag(
		"device",
		"Serial device",
	).Default("/dev/ttyUSB0").String()
	//useMock = kingpin.Flag(
	//	"use-mock",
	//	"Use mock data instead of a real device.",
	//).Bool()
	configFile = kingpin.Flag(
		"config",
		"[EXPERIMENTAL] Path to config yaml file that can enable TLS or authentication.",
	).Default("").String()

	powerConsumed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instantaneous_power_consumed",
		Help: "Instantaneous power consumed per phase in W",
	}, []string{"phase"})
	powerGenerated = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instantaneous_power_generated",
		Help: "Instantaneous power generated per phase in W",
	}, []string{"phase"})
	currentConsumed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instantaneous_current",
		Help: "Instantaneous current per phase in A",
	}, []string{"phase"})
	voltageConsumed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "instantaneous_voltage",
		Help: "Instantaneous voltage per phase in V",
	}, []string{"phase"})
	tariffIndicator = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tariff_indicator",
		Help: "Tariff indicator electricity",
	})
	electricityConsumed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "electricity_consumed",
		Help: "Electricity consumed per tariff in Wh",
	}, []string{"tariff"})
	electricityGenerated = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "electricity_generated",
		Help: "Electricity generated per tariff in Wh",
	}, []string{"tariff"})
	gasConsumed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gas_consumed",
		Help: "Gas consumed in m3",
	})

	logger log.Logger
)

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("node_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger = promlog.New(promlogConfig)

	go readFromP1()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>P1 Exporter</title></head>
			<body>
			<h1>P1 Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>`))
	})

	level.Info(logger).Log("msg", "Listening on", "address", *listenAddress)
	server := &http.Server{Addr: *listenAddress}

	if err := web.ListenAndServe(server, *configFile, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}

func floatValue(input string) (fval float64) {
	fval, _ = strconv.ParseFloat(input, 64)
	return
}

func readFromP1() {
	// open connection to serial port
	p1, _ := gop1.New(gop1.P1Config{
		USBDevice: *deviceName,
	})

	// start reading from P1 port
	p1.Start()

	for tgram := range p1.Incoming {
		for _, obj := range tgram.Objects {
			level.Debug(logger).Log(obj.Values[0].Value)

			switch obj.Type {

			case gop1.OBISTypeInstantaneousPowerDeliveredL1:
				powerConsumed.With(prometheus.Labels{"phase": "l1"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousPowerDeliveredL2:
				powerConsumed.With(prometheus.Labels{"phase": "l2"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousPowerDeliveredL3:
				powerConsumed.With(prometheus.Labels{"phase": "l3"}).Set(floatValue(obj.Values[0].Value))

			case gop1.OBISTypeInstantaneousPowerGeneratedL1:
				powerGenerated.With(prometheus.Labels{"phase": "l1"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousPowerGeneratedL2:
				powerGenerated.With(prometheus.Labels{"phase": "l2"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousPowerGeneratedL3:
				powerGenerated.With(prometheus.Labels{"phase": "l3"}).Set(floatValue(obj.Values[0].Value))

			case gop1.OBISTypeInstantaneousCurrentL1:
				currentConsumed.With(prometheus.Labels{"phase": "l1"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousCurrentL2:
				currentConsumed.With(prometheus.Labels{"phase": "l2"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousCurrentL3:
				currentConsumed.With(prometheus.Labels{"phase": "l3"}).Set(floatValue(obj.Values[0].Value))

			case gop1.OBISTypeInstantaneousVoltageL1:
				voltageConsumed.With(prometheus.Labels{"phase": "l1"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousVoltageL2:
				voltageConsumed.With(prometheus.Labels{"phase": "l2"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeInstantaneousVoltageL3:
				voltageConsumed.With(prometheus.Labels{"phase": "l3"}).Set(floatValue(obj.Values[0].Value))

			case gop1.OBISTypeElectricityTariffIndicator:
				tariffIndicator.Set(floatValue(obj.Values[0].Value))

			case gop1.OBISTypeElectricityDeliveredTariff1:
				electricityConsumed.With(prometheus.Labels{"tariff": "1"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeElectricityDeliveredTariff2:
				electricityConsumed.With(prometheus.Labels{"tariff": "2"}).Set(floatValue(obj.Values[0].Value))

			case gop1.OBISTypeElectricityGeneratedTariff1:
				electricityGenerated.With(prometheus.Labels{"tariff": "1"}).Set(floatValue(obj.Values[0].Value))
			case gop1.OBISTypeElectricityGeneratedTariff2:
				electricityGenerated.With(prometheus.Labels{"tariff": "2"}).Set(floatValue(obj.Values[0].Value))

			case gop1.OBISTypeGasDelivered:
				gasConsumed.Set(floatValue(obj.Values[1].Value))
			}
		}
	}
}
