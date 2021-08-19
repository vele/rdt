package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/kataras/tablewriter"
	"github.com/vele/rdt/vsphere"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	envURL      = "GOVMOMI_URL"
	envUserName = "GOVMOMI_USERNAME"
	envPassword = "GOVMOMI_PASSWORD"
	envInsecure = "GOVMOMI_INSECURE"
)

var urlDescription = fmt.Sprintf("ESX or vCenter URL [%s]", envURL)
var urlFlag = flag.String("url", getEnvString(envURL, ""), urlDescription)

var insecureDescription = fmt.Sprintf("Don't verify the server's certificate chain [%s]", envInsecure)
var networkDescription = fmt.Sprintf("List Networks ")
var insecureFlag = flag.Bool("insecure", getEnvBool(envInsecure, false), insecureDescription)
var flagNetwork = flag.Bool("network", false, networkDescription)
var interval = flag.Int("i", 20, "Interval ID")

func getEnvString(v string, def string) string {
	r := os.Getenv(v)
	if r == "" {
		return def
	}

	return r
}

func getEnvBool(v string, def bool) bool {
	r := os.Getenv(v)
	if r == "" {
		return def
	}

	switch strings.ToLower(r[0:1]) {
	case "t", "y", "1":
		return true
	}

	return false
}
func main() {
	flag.Parse()
	getInstances()
	//getInstancePerformanceMetrics()
}
func fetchNetworks() {

	Run(func(ctx context.Context, c *vim25.Client) error {
		// Create a view of Network types
		m := view.NewManager(c)

		v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Network"}, true)
		if err != nil {
			return err
		}

		defer v.Destroy(ctx)

		// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.Network.html
		var networks []mo.Network
		err = v.Retrieve(ctx, []string{"Network"}, nil, &networks)
		if err != nil {
			return err
		}

		for _, net := range networks {
			fmt.Printf("%s: %s\n", net.Name, net.Reference())
		}
		return nil
	})
}
func getInstances() {
	Run(func(ctx context.Context, c *vim25.Client) error {
		// Create view of VirtualMachine objects
		var data [][]string
		m := view.NewManager(c)

		v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
		if err != nil {
			return err
		}

		defer v.Destroy(ctx)

		var vms []mo.VirtualMachine
		err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary"}, &vms)
		if err != nil {
			return err
		}

		// Print summary per vm (see also: govc/vm/info.go)

		for _, vm := range vms {
			if vm.Summary.QuickStats.UptimeSeconds != 0 {
				data = append(data, []string{vm.Summary.Config.Name, strconv.FormatInt(int64(vm.Summary.Config.NumCpu), 10),
					strconv.FormatInt(int64(vm.Summary.Config.MemorySizeMB), 10), vm.Summary.Guest.IpAddress, strconv.FormatInt(int64(vm.Summary.QuickStats.UptimeSeconds/86400), 10), vm.Summary.})
			}

		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"VM Name", "Sign", "Rating"})

		for _, v := range data {
			table.Append(v)
		}

		table.Render() // Send output
		return nil
	})

}
func getNetworkCountersForInstance(instanceName string) []string {
	var data []string
	Run(func(ctx context.Context, c *vim25.Client) error {
		// Get virtual machines references
		m := view.NewManager(c)

		v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, nil, true)
		if err != nil {
			return err
		}

		defer v.Destroy(ctx)

		vmsRefs, err := v.Find(ctx, []string{"VirtualMachine"}, nil)
		if err != nil {
			return err
		}

		// Create a PerfManager
		perfManager := performance.NewManager(c)

		// Retrieve counters name list
		counters, err := perfManager.CounterInfoByName(ctx)
		if err != nil {
			return err
		}

		var names []string
		for name := range counters {
			if name == "net.usage.average" {
				names = append(names, name)
			}
		}

		spec := types.PerfQuerySpec{
			MaxSample:  1,
			MetricId:   []types.PerfMetricId{{Instance: "*"}},
			IntervalId: int32(*interval),
		}
		sample, err := perfManager.SampleByName(ctx, spec, names, vmsRefs)
		if err != nil {
			return err
		}
		result, err := perfManager.ToMetricSeries(ctx, sample)
		if err != nil {
			return err
		}

		for _, metric := range result {
			name := metric.Entity

			for _, v := range metric.Value {
				//counter := counters[v.Name]
				//units := counter.UnitInfo.GetElementDescription().Label

				instance := v.Instance
				if instance == "" {
					instance = "-"
				}

				if len(v.Value) != 0 {
					if name.String() == instanceName {
						if instance == "-" {
							data = append(data, v.ValueCSV())
						}
					}

				}
			}
		}
		return nil
	})
	return data
}
func Run(f func(ctx context.Context, c *vim25.Client) error) {
	var err error
	var c *vim25.Client
	parsedOptions := &vsphere.VsphereClientOptions{}
	parsedOptions.Url = *urlFlag
	parsedOptions.Insecure = *insecureFlag
	if *urlFlag == "" {
		err = simulator.VPX().Run(f)
	} else {
		ctx := context.Background()
		c, err = vsphere.NewClient(ctx, parsedOptions)
		if err == nil {
			err = f(ctx, c)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
