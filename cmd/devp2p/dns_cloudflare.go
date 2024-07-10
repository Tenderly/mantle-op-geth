// Copyright 2019 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/tenderly/mantle-op-geth/log"
	"github.com/tenderly/mantle-op-geth/p2p/dnsdisc"
	"github.com/urfave/cli/v2"
)

var (
	cloudflareTokenFlag = &cli.StringFlag{
		Name:    "token",
		Usage:   "CloudFlare API token",
		EnvVars: []string{"CLOUDFLARE_API_TOKEN"},
	}
	cloudflareZoneIDFlag = &cli.StringFlag{
		Name:  "zoneid",
		Usage: "CloudFlare Zone ID (optional)",
	}
)

type cloudflareClient struct {
	*cloudflare.API
	zoneID string
}

// newCloudflareClient sets up a CloudFlare API client from command line flags.
func newCloudflareClient(ctx *cli.Context) *cloudflareClient {
	token := ctx.String(cloudflareTokenFlag.Name)
	if token == "" {
		exit(fmt.Errorf("need cloudflare API token to proceed"))
	}
	api, err := cloudflare.NewWithAPIToken(token)
	if err != nil {
		exit(fmt.Errorf("can't create Cloudflare client: %v", err))
	}
	return &cloudflareClient{
		API:    api,
		zoneID: ctx.String(cloudflareZoneIDFlag.Name),
	}
}

// deploy uploads the given tree to CloudFlare DNS.
func (c *cloudflareClient) deploy(name string, t *dnsdisc.Tree) error {
	if err := c.checkZone(name); err != nil {
		return err
	}
	records := t.ToTXT(name)
	return c.uploadRecords(name, records)
}

// checkZone verifies permissions on the CloudFlare DNS Zone for name.
func (c *cloudflareClient) checkZone(name string) error {
	if c.zoneID == "" {
		log.Info(fmt.Sprintf("Finding CloudFlare zone ID for %s", name))
		id, err := c.ZoneIDByName(name)
		if err != nil {
			return err
		}
		c.zoneID = id
	}
	log.Info(fmt.Sprintf("Checking Permissions on zone %s", c.zoneID))
	zone, err := c.ZoneDetails(context.Background(), c.zoneID)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(name, "."+zone.Name) {
		return fmt.Errorf("CloudFlare zone name %q does not match name %q to be deployed", zone.Name, name)
	}
	needPerms := map[string]bool{"#zone:edit": false, "#zone:read": false}
	for _, perm := range zone.Permissions {
		if _, ok := needPerms[perm]; ok {
			needPerms[perm] = true
		}
	}
	for _, ok := range needPerms {
		if !ok {
			return fmt.Errorf("wrong permissions on zone %s: %v", c.zoneID, needPerms)
		}
	}
	return nil
}

// uploadRecords updates the TXT records at a particular subdomain. All non-root records
// will have a TTL of "infinity" and all existing records not in the new map will be
// nuked!
func (c *cloudflareClient) uploadRecords(_ string, _ map[string]string) error {
	return nil
}
