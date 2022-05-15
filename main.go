package main

import (
	"encoding/binary"
	"errors"
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
	promVersion.Version = "0.1.1"
	prometheus.MustRegister(promVersion.NewCollector("pv_modbus_exporter"))
}

func main() {
	var (
		listenAddr   = flag.String("web.listen-address", ":9502", "The address to listen on for HTTP requests.")
		inverterAddr = flag.String("inverter.addr", "localhost:502", "The IP address of the PV invester.")
		pollInterval = flag.Int("inverter.poll-interval", 5, "Interval in seconds between polls.")
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

	// Poll inverter values
	go func() {

		// Establish inverter modbus connection
		handler := modbus.NewTCPClientHandler(*inverterAddr)
		handler.SlaveId = byte(*slaveId)
		err := handler.Connect()
		if err != nil {
			log.Printf("Error: %s", err)
			os.Exit(1)
		}

		defer handler.Close()
		client := modbus.NewClient(handler)

		for {
			// Daily yield
			res, err := readRegisters(client, 30517, 4)
			if err != nil {
				log.Printf("Error: %s", err)
				os.Exit(1)
			}
			pvDailyYield.Set(res)

			// MPPT1
			res, err = readRegisters(client, 30773, 2)
			if err != nil {
				log.Printf("Error: %s", err)
				os.Exit(1)
			}
			pvMppt1Watts.Set(res)

			// MPPT2
			res, err = readRegisters(client, 30961, 2)
			if err != nil {
				log.Printf("Error: %s", err)
				os.Exit(1)
			}
			pvMppt2Watts.Set(res)

			// Total
			res, err = readRegisters(client, 30775, 2)
			if err != nil {
				log.Printf("Error: %s", err)
				os.Exit(1)
			}
			pvTotalWatts.Set(res)

			time.Sleep(time.Duration(*pollInterval) * time.Second)
		}
	}()

	// Expose the registered metrics via HTTP
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{},
	))
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}

func readRegisters(client modbus.Client, no uint16, size uint16) (float64, error) {
	results, err := client.ReadInputRegisters(no, size)
	if err != nil {
		return 0, err
	}

	switch size {
	case 4:
		return float64(binary.BigEndian.Uint64(results)), nil
	case 2:
		// Ugly hack: Instead of no value or zero, result contains [128 0 0 0], resp. 2147483648
		res := binary.BigEndian.Uint32(results)
		if res == 2147483648 {
			return float64(0), nil
		} else {
			return float64(res), nil
		}
	case 1:
		return float64(binary.BigEndian.Uint16(results)), nil
	default:
		return 0, errors.New("Error: Invalid size")
	}
}
