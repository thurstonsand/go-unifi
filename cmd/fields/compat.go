package main

import (
	"github.com/ubiquiti-community/go-unifi/internal/fields"
)

// addSettingCompatFields injects setting fields the JAR field spec no longer
// describes but that live controllers still store and honor. Without them the
// generated models drop the values on every write.
func addSettingCompatFields(resource *ResourceInfo, baseType *FieldInfo) {
	if resource.StructName == "SettingUsg" {
		// Removed in v7, retaining for backwards compatibility
		baseType.Fields["MdnsEnabled"] = NewFieldInfo("MdnsEnabled", "mdns_enabled", fields.Bool, "", false, false, false, "")
		// The 10.x spec moved GeoIP filtering to its own ip_filtering payload,
		// but the usg setting still carries these fields on the wire and the
		// controller keeps honoring them.
		baseType.Fields["GeoIPFilteringBlock"] = NewFieldInfo("GeoIPFilteringBlock", "geo_ip_filtering_block", fields.String, "block|allow", true, false, false, "")
		baseType.Fields["GeoIPFilteringCountries"] = NewFieldInfo("GeoIPFilteringCountries", "geo_ip_filtering_countries", fields.String, `^([A-Z]{2})?(,[A-Z]{2}){0,149}$`, true, false, false, "")
		baseType.Fields["GeoIPFilteringEnabled"] = NewFieldInfo("GeoIPFilteringEnabled", "geo_ip_filtering_enabled", fields.Bool, "", false, false, false, "")
		baseType.Fields["GeoIPFilteringTrafficDirection"] = NewFieldInfo("GeoIPFilteringTrafficDirection", "geo_ip_filtering_traffic_direction", fields.String, `^(both|ingress|egress)$`, true, false, false, "")
	}
	if resource.StructName == "SettingIps" {
		// The 10.5.x field spec moved IPS alert suppression out of the ips
		// setting and into a standalone ips_suppression setting. Controllers
		// still store and echo the same payload nested under the ips setting's
		// "suppression" field, and controllers older than 10.4 only accept it
		// there, so keep modelling the nested shape as well. Nested types are
		// named after the owning resource plus the field, so these come out as
		// SettingIpsAlerts/Tracking/Whitelist, distinct from the standalone
		// setting's SettingIpsSuppression* types.
		for typeName, jsonName := range map[string]string{
			resource.StructName + "Tracking":  "tracking",
			resource.StructName + "Whitelist": "whitelist",
		} {
			entry := NewFieldInfo(typeName, jsonName, "struct", "", false, false, false, "")
			entry.Fields = map[string]*FieldInfo{
				"Direction": NewFieldInfo("Direction", "direction", fields.String, "both|src|dest", true, false, false, ""),
				"Mode":      NewFieldInfo("Mode", "mode", fields.String, "ip|subnet|network", true, false, false, ""),
				"Value":     NewFieldInfo("Value", "value", fields.String, "", true, false, false, ""),
			}
			resource.Types[typeName] = entry
		}

		alerts := NewFieldInfo(resource.StructName+"Alerts", "alerts", "struct", "", false, false, false, "")
		alerts.Fields = map[string]*FieldInfo{
			"Category":  NewFieldInfo("Category", "category", fields.String, "", true, false, false, ""),
			"Gid":       NewFieldInfo("Gid", "gid", fields.Int, "", true, false, true, fields.Number),
			"ID":        NewFieldInfo("ID", "id", fields.Int, "", true, false, true, fields.Number),
			"Signature": NewFieldInfo("Signature", "signature", fields.String, "", true, false, false, ""),
			"Tracking":  NewFieldInfo("Tracking", "tracking", resource.StructName+"Tracking", "", true, true, false, ""),
			"Type":      NewFieldInfo("Type", "type", fields.String, "all|track", true, false, false, ""),
		}
		resource.Types[alerts.FieldName] = alerts

		suppression := NewFieldInfo(resource.StructName+"Suppression", "suppression", "struct", "", false, false, false, "")
		suppression.Fields = map[string]*FieldInfo{
			"Alerts":    NewFieldInfo("Alerts", "alerts", alerts.FieldName, "", true, true, false, ""),
			"Whitelist": NewFieldInfo("Whitelist", "whitelist", resource.StructName+"Whitelist", "", true, true, false, ""),
		}
		resource.Types[suppression.FieldName] = suppression

		baseType.Fields["Suppression"] = NewFieldInfo("Suppression", "suppression", suppression.FieldName, "", true, false, true, "")
	}
	if resource.StructName == "SettingMdns" {
		// The 10.5.x field spec describes only mode and the service lists, but
		// a Network 10.5.67 controller stores the scope of mDNS repeating in
		// enabled_for ("all"/"some") plus enabled_for_network_ids, and drops
		// the scope when a write omits them. Model both so callers can round-
		// trip custom mode.
		baseType.Fields["EnabledFor"] = NewFieldInfo("EnabledFor", "enabled_for", fields.String, "", true, false, false, "")
		baseType.Fields["EnabledForNetworkIDs"] = NewFieldInfo("EnabledForNetworkIDs", "enabled_for_network_ids", fields.String, "", false, true, false, "")
		// The service lists must stay serialized when empty: the default
		// codegen marks slices omitempty, which drops an explicit [] and makes
		// clearing every predefined or custom service a silent no-op.
		resource.FieldProcessor = func(name string, f *FieldInfo) error {
			switch name {
			case "CustomServices", "PredefinedServices":
				f.OmitEmpty = false
			}
			return nil
		}
	}
}

// addNonSpecNestedFields re-adds fields the JAR schema does not describe to
// nested types. Unlike the top-level injections in NewResource, these types only
// exist once the spec has been processed.
func (r *ResourceInfo) addNonSpecNestedFields() {
	if r.StructName != "Device" {
		return
	}
	if portOverrides, ok := r.Types["DevicePortOverrides"]; ok {
		portOverrides.Fields["TaggedNetworkIDs"] = NewFieldInfo("TaggedNetworkIDs", "tagged_networkconf_ids", fields.String, "", true, true, false, "")
	}
	if radioTable, ok := r.Types["DeviceRadioTable"]; ok {
		// 802.11k assisted roaming left the 10.x radio_table spec, but access
		// points still accept and report both fields; dropping them from the
		// model discards the setting on every device write.
		radioTable.Fields["AssistedRoamingEnabled"] = NewFieldInfo("AssistedRoamingEnabled", "assisted_roaming_enabled", fields.Bool, "", true, false, false, "")
		radioTable.Fields["AssistedRoamingRssi"] = NewFieldInfo("AssistedRoamingRssi", "assisted_roaming_rssi", fields.Int, `^-([6-7][0-9]|80)$`, true, false, true, fields.Number)
	}
}
