// +build !nonetdev

package collector

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

/*
#cgo CFLAGS: -D_IFI_OQDROPS
#include <stdio.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <ifaddrs.h>
#include <net/if.h>
*/
import "C"

const (
	subsystem  = "network"
)

type netDevCollector struct {
	metricDescs map[string]*prometheus.Desc
}

func init() {
	Factories["netdev"] = NewNetDevCollector
}

// Takes a prometheus registry and returns a new Collector exposing
// Network device stats.
func NewNetDevCollector() (Collector, error) {
	return &netDevCollector{
		metricDescs: map[string]*prometheus.Desc{},
	}, nil
}

func (c *netDevCollector) Update(ch chan<- prometheus.Metric) (err error) {
	netDev, err := getNetDevStats()
	if err != nil {
		return fmt.Errorf("couldn't get netstats: %s", err)
	}
	for dev, devStats := range netDev {
		for key, value := range devStats {
                        desc, ok := c.metricDescs[key]
                        if !ok {
                                desc = prometheus.NewDesc(
                                        prometheus.BuildFQName(
                                                Namespace, subsystem, key),
                                        fmt.Sprintf(
                                                "%s from getifaddrs().", key),
                                        []string{"device"},
                                        nil,
                                )
                                c.metricDescs[key] = desc
                        }
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf(
					"Invalid value %s in netstats: %s",
					value, err)
			}
			ch <- prometheus.MustNewConstMetric(
				desc, prometheus.GaugeValue, v, dev)
		}
	}
	return nil
}

func getNetDevStats() (map[string]map[string]string, error) {
	netDev := map[string]map[string]string{}

	var ifap, ifa *C.struct_ifaddrs
	if C.getifaddrs(&ifap) == -1 {
		return nil, errors.New("getifaddrs() failed")
	}
	defer C.freeifaddrs(ifap)

	for ifa = ifap; ifa != nil; ifa = ifa.ifa_next {
		if ifa.ifa_addr.sa_family == C.AF_LINK {
			devStats := map[string]string{}
			data := (*C.struct_if_data)(ifa.ifa_data)

			devStats["receive_packets"] = strconv.Itoa(int(data.ifi_ipackets))
			devStats["transmit_packets"] = strconv.Itoa(int(data.ifi_opackets))
			devStats["receive_errs"] = strconv.Itoa(int(data.ifi_ierrors))
			devStats["transmit_errs"] = strconv.Itoa(int(data.ifi_oerrors))
			devStats["receive_bytes"] = strconv.Itoa(int(data.ifi_ibytes))
			devStats["transmit_bytes"] = strconv.Itoa(int(data.ifi_obytes))
			devStats["receive_multicast"] = strconv.Itoa(int(data.ifi_imcasts))
			devStats["transmit_multicast"] = strconv.Itoa(int(data.ifi_omcasts))
			devStats["receive_drop"] = strconv.Itoa(int(data.ifi_iqdrops))
			devStats["transmit_drop"] = strconv.Itoa(int(data.ifi_oqdrops))
                        netDev[C.GoString(ifa.ifa_name)] = devStats
		}
	}

	return netDev, nil
}
