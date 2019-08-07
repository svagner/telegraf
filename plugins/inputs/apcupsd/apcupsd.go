package apcupsd

import (
	"context"
	"net/url"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/mdlayher/apcupsd"
)

const defaultAddress = "tcp://127.0.0.1:3551"

var defaultTimeout = internal.Duration{Duration: time.Duration(time.Second * 5)}

type ApcUpsd struct {
	Servers []string
	Timeout internal.Duration
}

func (*ApcUpsd) Description() string {
	return "Monitor APC UPSes connected to apcupsd"
}

var sampleConfig = `
  # A list of running apcupsd server to connect to.
  # If not provided will default to tcp://127.0.0.1:3551
  servers = ["tcp://127.0.0.1:3551"]

  ## Timeout for dialing server.
  timeout = "5s"
`

func (*ApcUpsd) SampleConfig() string {
	return sampleConfig
}

func (h *ApcUpsd) Gather(acc telegraf.Accumulator) error {
	ctx := context.Background()

	for _, addr := range h.Servers {
		addrBits, err := url.Parse(addr)
		if err != nil {
			return err
		}
		if addrBits.Scheme == "" {
			addrBits.Scheme = "tcp"
		}

		ctx, cancel := context.WithTimeout(ctx, h.Timeout.Duration)
		defer cancel()

		status, err := fetchStatus(ctx, addrBits)
		if err != nil {
			return err
		}

		tags := map[string]string{
			"serial":   status.SerialNumber,
			"ups_name": status.UPSName,
			"status":   status.Status,
		}

		fields := map[string]interface{}{
			"status_online":          boolToInt(status.Status == "ONLINE"), // todo: revisit
			"input_voltage":          status.LineVoltage,
			"load_percent":           status.LoadPercent,
			"battery_charge_percent": status.BatteryChargePercent,
			"time_left_ns":           status.TimeLeft.Nanoseconds(),
			"output_voltage":         status.OutputVoltage,
			"internal_temp":          status.InternalTemp,
			"battery_voltage":        status.BatteryVoltage,
			"input_frequency":        status.LineFrequency,
			"time_on_battery_ns":     status.TimeOnBattery.Nanoseconds(),
		}

		acc.AddFields("apcupsd", fields, tags)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func fetchStatus(ctx context.Context, addr *url.URL) (*apcupsd.Status, error) {
	client, err := apcupsd.DialContext(ctx, addr.Scheme, addr.Host)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return client.Status()
}

func init() {
	inputs.Add("apcupsd", func() telegraf.Input {
		return &ApcUpsd{
			Servers: []string{defaultAddress},
			Timeout: defaultTimeout,
		}
	})
}
