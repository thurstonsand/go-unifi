package unifi

import (
	"context"
	"fmt"
	"net/http"
)

// This is a v2 API object, manually coded.

// GlobalNetworkConfig is the site-wide network configuration served by the v2
// API. It owns the authoritative per-network membership lists for mDNS and
// IGMP snooping.
//
// The legacy documents carry read-only projections of the mDNS membership:
// `setting/mdns.enabled_for_network_ids` and, per network,
// `networkconf.mdns_enabled`. The controller rewrites both whenever this object
// changes. Writing either projection directly is accepted with rc "ok" and then
// discarded, so mDNS membership must be changed here.
type GlobalNetworkConfig struct {
	DefaultSecurityPosture             string   `json:"default_security_posture,omitempty"`
	FloodUnknownMulticastForNetworkIDs []string `json:"flood_unknown_multicast_for_network_ids,omitempty"`
	IGMPFastleaveForNetworkIDs         []string `json:"igmp_fastleave_for_network_ids,omitempty"`
	IGMPSnoopingFor                    string   `json:"igmp_snooping_for,omitempty"`
	IGMPSnoopingForNetworkIDs          []string `json:"igmp_snooping_for_network_ids,omitempty"`
	MdnsEnabledFor                     string   `json:"mdns_enabled_for,omitempty"`
	MdnsEnabledForNetworkIDs           []string `json:"mdns_enabled_for_network_ids,omitempty"`
}

// mdnsMembership is the write half of GlobalNetworkConfig. The endpoint merges
// the fields it is given, so sending only these two leaves the IGMP and IPv6
// members of the same object untouched.
type mdnsMembership struct {
	MdnsEnabledFor           string   `json:"mdns_enabled_for"`
	MdnsEnabledForNetworkIDs []string `json:"mdns_enabled_for_network_ids"`
}

func (c *ApiClient) GetGlobalNetworkConfig(
	ctx context.Context,
	site string,
) (*GlobalNetworkConfig, error) {
	var respBody GlobalNetworkConfig

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/global/config/network", site),
		nil,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

// SetMdnsMembership points mDNS at the given networks and returns the whole
// updated configuration.
//
// enabledFor mirrors the UI's scope control: "all" reflects on every network,
// "some" restricts to networkIDs, and "none" disables reflection.
func (c *ApiClient) SetMdnsMembership(
	ctx context.Context,
	site string,
	enabledFor string,
	networkIDs []string,
) (*GlobalNetworkConfig, error) {
	var respBody GlobalNetworkConfig

	if networkIDs == nil {
		networkIDs = []string{}
	}

	body := mdnsMembership{
		MdnsEnabledFor:           enabledFor,
		MdnsEnabledForNetworkIDs: networkIDs,
	}

	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("v2/api/site/%s/global/config/network", site),
		body,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
