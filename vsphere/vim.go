package vsphere

import (
	"context"

	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
)

type VsphereClientOptions struct {
	Url      string
	Insecure bool
}

func NewClient(ctx context.Context, v *VsphereClientOptions) (*vim25.Client, error) {
	u, err := soap.ParseURL(v.Url)
	if err != nil {
		return nil, err
	}
	processOverride(u)
	s := &cache.Session{
		URL:      u,
		Insecure: v.Insecure,
	}
	c := new(vim25.Client)
	err = s.Login(ctx, c, nil)
	if err != nil {
		return nil, err
	}
	return c, nil

}
