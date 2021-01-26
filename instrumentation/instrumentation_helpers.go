package instrumentation

import (
	"os"
	"runtime/debug"
	"time"

	"github.com/TykTechnologies/tyk-pump/config"
	"github.com/TykTechnologies/tyk-pump/logger"
	"github.com/TykTechnologies/tyk/rpc"

	"github.com/gocraft/health"
)

var applicationGCStats = debug.GCStats{}
var Instrument = health.NewStream()
var log = logger.GetLogger()

// SetupInstrumentation handles all the intialisation of the instrumentation handler
func SetupInstrumentation(config *config.TykPumpConfiguration) {
	var enabled bool
	//Instrument.AddSink(&health.WriterSink{os.Stdout})
	thisInstr := os.Getenv("TYK_INSTRUMENTATION")

	if thisInstr == "1" {
		enabled = true
	}

	if !enabled {
		return
	}

	if config.StatsdConnectionString == "" {
		log.Error("Instrumentation is enabled, but no connectionstring set for statsd")
		return
	}

	log.Info("Sending stats to: ", config.StatsdConnectionString, " with prefix: ", config.StatsdPrefix)
	statsdSink, err := NewStatsDSink(config.StatsdConnectionString,
		&StatsDSinkOptions{Prefix: config.StatsdPrefix})

	if err != nil {
		log.Fatal("Failed to start StatsD check: ", err)
		return
	}

	log.Info("StatsD instrumentation sink started")
	Instrument.AddSink(statsdSink)

	rpc.Instrument = Instrument

	MonitorApplicationInstrumentation()
}

func MonitorApplicationInstrumentation() {
	log.Info("Starting application monitoring...")
	go func() {
		job := Instrument.NewJob("GCActivity")
		metadata := health.Kvs{"host": "pump"}
		applicationGCStats.PauseQuantiles = make([]time.Duration, 5)

		for {
			debug.ReadGCStats(&applicationGCStats)
			job.GaugeKv("pauses_quantile_min", float64(applicationGCStats.PauseQuantiles[0].Nanoseconds()), metadata)
			job.GaugeKv("pauses_quantile_25", float64(applicationGCStats.PauseQuantiles[1].Nanoseconds()), metadata)
			job.GaugeKv("pauses_quantile_50", float64(applicationGCStats.PauseQuantiles[2].Nanoseconds()), metadata)
			job.GaugeKv("pauses_quantile_75", float64(applicationGCStats.PauseQuantiles[3].Nanoseconds()), metadata)
			job.GaugeKv("pauses_quantile_max", float64(applicationGCStats.PauseQuantiles[4].Nanoseconds()), metadata)

			time.Sleep(5 * time.Second)
		}
	}()
}
