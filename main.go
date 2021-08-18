package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/vele/rdt/vsphere"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
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
var insecureFlag = flag.Bool("insecure", getEnvBool(envInsecure, false), insecureDescription)

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
