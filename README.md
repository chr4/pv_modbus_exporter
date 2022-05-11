# Modbus/TCP inverter Prometheus exporter

This Prometheus exporter connects to a photovoltaik inverter and retrieves data via Modbus/TCP.

It's currently retrieving the following values:

- Current watts (MPPT1)
- Current watts (MPPT2)
- Total current watts
- Daily yield in kWh

Should work with SMA Tripower inverters, can be easily adapted for other values and other compatible inverters.
