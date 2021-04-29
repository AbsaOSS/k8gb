/*
Copyright 2021 The k8gb Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Generated by GoLic, for more details see: https://github.com/AbsaOSS/golic
*/
package dns

import (
	"fmt"
	"strconv"

	"github.com/AbsaOSS/k8gb/controllers/depresolver"

	ibclient "github.com/infobloxopen/infoblox-go-client"
)

type infobloxClient struct {
	config     depresolver.Config
	connection *ibclient.Connector
}

func newInfobloxClient(config depresolver.Config) *infobloxClient {
	return &infobloxClient{
		config: config,
	}
}

func (c *infobloxClient) login() (objectManager *ibclient.ObjectManager, err error) {
	const cmdType = "k8gbclient"
	hostConfig := ibclient.HostConfig{
		Host:     c.config.Infoblox.Host,
		Version:  c.config.Infoblox.Version,
		Port:     strconv.Itoa(c.config.Infoblox.Port),
		Username: c.config.Infoblox.Username,
		Password: c.config.Infoblox.Password,
	}
	transportConfig := ibclient.NewTransportConfig("false", c.config.Infoblox.HTTPRequestTimeout, c.config.Infoblox.HTTPPoolConnections)
	requestBuilder := &ibclient.WapiRequestBuilder{}
	requestor := &ibclient.WapiHttpRequestor{}

	if c.config.Override.FakeInfobloxEnabled {
		fqdn := "fakezone.example.com"
		fakeRefReturn := "zone_delegated/ZG5zLnpvbmUkLl9kZWZhdWx0LnphLmNvLmFic2EuY2Fhcy5vaG15Z2xiLmdzbGJpYmNsaWVudA:fakezone.example.com/default"
		k8gbFakeConnector := &fakeInfobloxConnector{
			getObjectObj: ibclient.NewZoneDelegated(ibclient.ZoneDelegated{Fqdn: fqdn}),
			getObjectRef: "",
			resultObject: []ibclient.ZoneDelegated{*ibclient.NewZoneDelegated(ibclient.ZoneDelegated{Fqdn: fqdn, Ref: fakeRefReturn})},
		}
		objectManager = ibclient.NewObjectManager(k8gbFakeConnector, cmdType, "")
	} else {
		c.connection, err = ibclient.NewConnector(hostConfig, transportConfig, requestBuilder, requestor)
		if err != nil {
			return
		}
		objectManager = ibclient.NewObjectManager(c.connection, cmdType, "")
	}
	return
}

func (c *infobloxClient) logout() {
	if c.config.Override.FakeInfobloxEnabled {
		return
	}
	log.Debug().Msgf("Infoblox logout")
	err := c.connection.Logout()
	if err != nil {
		log.Err(err).Msg("Failed to close connection to infoblox")
	}
}

func (c *infobloxClient) checkZoneDelegated(findZone *ibclient.ZoneDelegated) (err error) {
	if findZone.Fqdn != c.config.DNSZone {
		return fmt.Errorf("delegated zone returned from infoblox(%s) does not match requested gslb zone(%s)", findZone.Fqdn, c.config.DNSZone)
	}
	return
}

func (c *infobloxClient) filterOutDelegateTo(delegateTo []ibclient.NameServer, fqdn string) []ibclient.NameServer {
	for i := 0; i < len(delegateTo); i++ {
		if delegateTo[i].Name == fqdn {
			delegateTo = append(delegateTo[:i], delegateTo[i+1:]...)
			i--
		}
	}
	return delegateTo
}
