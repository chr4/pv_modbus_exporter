package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promVersion "github.com/prometheus/common/version"

	"github.com/goburrow/modbus"
)

func init() {
	promVersion.Version = "0.1.0"
	prometheus.MustRegister(promVersion.NewCollector("pv_modbus_exporter"))
}

func main() {
	var (
		listenAddr   = flag.String("web.listen-address", ":9502", "The address to listen on for HTTP requests.")
		inverterAddr = flag.String("inverter.addr", "localhost:502", "The IP address of the PV invester.")
		slaveId      = flag.Int("inverter.slave-id", 3, "The modbus slave ID to use")
		showVersion  = flag.Bool("version", false, "Print version information and exit.")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("%s\n", promVersion.Print("pv_modbus_exporter"))
		os.Exit(0)
	}

	var pvMppt1Watts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pv_mppt1_watts",
		Help: "PV inverter current power (MPPT 1)",
	})
	var pvMppt2Watts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pv_mppt2_watts",
		Help: "PV inverter current power (MPPT 2)",
	})
	var pvTotalWatts = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pv_total_watts",
		Help: "PV inverter current power (all MPPTs combined)",
	})
	var pvDailyYield = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pv_daily_yield",
		Help: "PV daily yield (up until now)",
	})

	// Register the summary and the histogram with Prometheus's default registry
	prometheus.MustRegister(pvMppt1Watts)
	prometheus.MustRegister(pvMppt2Watts)
	prometheus.MustRegister(pvTotalWatts)
	prometheus.MustRegister(pvDailyYield)

	// Add Go module build info
	prometheus.MustRegister(collectors.NewBuildInfoCollector())

	// Establish inverter modbus connection
	handler := modbus.NewTCPClientHandler(*inverterAddr)
	handler.SlaveId = byte(*slaveId)
	err := handler.Connect()
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}

	defer handler.Close()

	client := modbus.NewClient(handler)

	// Poll inverter values
	go func() {
		for {
			// Daily yield
			results, err := client.ReadInputRegisters(30517, 4)
			if err != nil {
				log.Printf("Error: %s", err)
				return
			}
			pvDailyYield.Set(float64(binary.BigEndian.Uint64(results)))

			// MPPT1
			results, err = client.ReadInputRegisters(30773, 2)
			if err != nil {
				log.Printf("Error: %s", err)
				return
			}
			pvMppt1Watts.Set(float64(binary.BigEndian.Uint32(results)))

			// MPPT2
			results, err = client.ReadInputRegisters(30961, 2)
			if err != nil {
				log.Printf("Error: %s", err)
				return
			}
			pvMppt2Watts.Set(float64(binary.BigEndian.Uint32(results)))

			// Total
			results, err = client.ReadInputRegisters(30775, 2)
			if err != nil {
				log.Printf("Error: %s", err)
				return
			}
			pvTotalWatts.Set(float64(binary.BigEndian.Uint32(results)))

			time.Sleep(1)
		}
	}()

	// Expose the registered metrics via HTTP
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
