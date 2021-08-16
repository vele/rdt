package vsphere

import (
	"context"
	"log"

	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

const (
	envURL      = "GOVMOMI_URL"
	envUserName = "GOVMOMI_USERNAME"
	envPassword = "GOVMOMI_PASSWORD"
	envInsecure = "GOVMOMI_INSECURE"
)

type VsphereClientOptions struct {
	urlFlag      string
	insecureFlag bool
}

func (v VsphereClientOptions) NewClient(ctx context.Context) (*vim25.Client, error) {
	u, err := soap.ParseURL(v.urlFlag)
	if err != nil {
		return nil, err
	}
	processOverride(u)
	s := &cache.Session{
		URL:      u,
		Insecure: v.insecureFlag,
	}
	c := new(vim25.Client)
	err = s.Login(ctx, c, nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}
func (v VsphereClientOptions) Run(f func(context.Context, *vim25.Client) error) {

	var err error
	var c *vim25.Client

	if v.urlFlag == "" {
		err = simulator.VPX().Run(f)
	} else {
		ctx := context.Background()
		c, err = v.NewClient(ctx)
		if err == nil {
			err = f(ctx, c)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
