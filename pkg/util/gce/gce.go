// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package gce

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/common"
	httputils "github.com/DataDog/datadog-agent/pkg/util/http"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// declare these as vars not const to ease testing
var (
	metadataURL = "http://169.254.169.254/computeMetadata/v1"

	// CloudProviderName contains the inventory name of for EC2
	CloudProviderName = "GCP"
)

// IsRunningOn returns true if the agent is running on GCE
func IsRunningOn() bool {
	if _, err := GetHostname(); err == nil {
		return true
	}
	return false
}

// GetHostname returns the hostname querying GCE Metadata api
func GetHostname() (string, error) {
	if !config.IsCloudProviderEnabled(CloudProviderName) {
		return "", fmt.Errorf("cloud provider is disabled by configuration")
	}
	hostname, err := getResponseWithMaxLength(metadataURL+"/instance/hostname",
		config.Datadog.GetInt("metadata_endpoints_max_hostname_size"))
	if err != nil {
		return "", fmt.Errorf("unable to retrieve hostname from GCE: %s", err)
	}
	return hostname, nil
}

// GetHostAliases returns the host aliases from GCE
func GetHostAliases() ([]string, error) {
	if !config.IsCloudProviderEnabled(CloudProviderName) {
		return nil, fmt.Errorf("cloud provider is disabled by configuration")
	}

	aliases := []string{}

	hostname, err := GetHostname()
	if err == nil {
		aliases = append(aliases, hostname)
	} else {
		log.Debugf("failed to get hostname to use as Host Alias: %s", err)
	}

	if instanceAlias, err := getInstanceAlias(hostname); err == nil {
		aliases = append(aliases, instanceAlias)
	} else {
		log.Debugf("failed to get Host Alias: %s", err)
	}

	return aliases, nil
}

func getInstanceAlias(hostname string) (string, error) {
	instanceName, err := getResponseWithMaxLength(metadataURL+"/instance/name",
		config.Datadog.GetInt("metadata_endpoints_max_hostname_size"))
	if err != nil {
		// If the endpoint is not reachable, fallback on the old way to get the alias.
		// For instance, it happens in GKE, where the metadata server is only a subset
		// of the Compute Engine metadata server.
		// See https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity#gke_mds
		if hostname == "" {
			return "", fmt.Errorf("unable to retrieve instance name and hostname from GCE: %s", err)
		}
		instanceName = strings.SplitN(hostname, ".", 2)[0]
	}

	projectID, err := getResponseWithMaxLength(metadataURL+"/project/project-id",
		config.Datadog.GetInt("metadata_endpoints_max_hostname_size"))
	if err != nil {
		return "", fmt.Errorf("unable to retrieve project ID from GCE: %s", err)
	}
	return fmt.Sprintf("%s.%s", instanceName, projectID), nil
}

// GetClusterName returns the name of the cluster containing the current GCE instance
func GetClusterName() (string, error) {
	if !config.IsCloudProviderEnabled(CloudProviderName) {
		return "", fmt.Errorf("cloud provider is disabled by configuration")
	}
	clusterName, err := getResponseWithMaxLength(metadataURL+"/instance/attributes/cluster-name",
		config.Datadog.GetInt("metadata_endpoints_max_hostname_size"))
	if err != nil {
		return "", fmt.Errorf("unable to retrieve clustername from GCE: %s", err)
	}
	return clusterName, nil
}

// GetPublicIPv4 returns the public IPv4 address of the current GCE instance
func GetPublicIPv4() (string, error) {
	if !config.IsCloudProviderEnabled(CloudProviderName) {
		return "", fmt.Errorf("cloud provider is disabled by configuration")
	}
	publicIPv4, err := getResponseWithMaxLength(metadataURL+"/instance/network-interfaces/0/access-configs/0/external-ip",
		config.Datadog.GetInt("metadata_endpoints_max_hostname_size"))
	if err != nil {
		return "", fmt.Errorf("unable to retrieve public IPv4 from GCE: %s", err)
	}
	return publicIPv4, nil
}

// GetNetworkID retrieves the network ID using the metadata endpoint. For
// GCE instances, the the network ID is the VPC ID, if the instance is found to
// be a part of exactly one VPC.
func GetNetworkID() (string, error) {
	if !config.IsCloudProviderEnabled(CloudProviderName) {
		return "", fmt.Errorf("cloud provider is disabled by configuration")
	}
	resp, err := getResponse(metadataURL + "/instance/network-interfaces/")
	if err != nil {
		return "", fmt.Errorf("unable to retrieve network-interfaces from GCE: %s", err)
	}

	interfaceIDs := strings.Split(strings.TrimSpace(resp), "\n")
	vpcIDs := common.NewStringSet()

	for _, interfaceID := range interfaceIDs {
		if interfaceID == "" {
			continue
		}
		interfaceID = strings.TrimSuffix(interfaceID, "/")
		id, err := getResponse(metadataURL + fmt.Sprintf("/instance/network-interfaces/%s/network", interfaceID))
		if err != nil {
			return "", err
		}
		vpcIDs.Add(id)
	}

	switch len(vpcIDs) {
	case 0:
		return "", fmt.Errorf("zero network interfaces detected")
	case 1:
		return vpcIDs.GetAll()[0], nil
	default:
		return "", fmt.Errorf("more than one network interface detected, cannot get network ID")
	}

}

// GetNTPHosts returns the NTP hosts for GCE if it is detected as the cloud provider, otherwise an empty array.
// Docs: https://cloud.google.com/compute/docs/instances/managing-instances
func GetNTPHosts() []string {
	if IsRunningOn() {
		return []string{"metadata.google.internal"}
	}

	return nil
}

func getResponseWithMaxLength(endpoint string, maxLength int) (string, error) {
	result, err := getResponse(endpoint)
	if err != nil {
		return result, err
	}
	if len(result) > maxLength {
		return "", fmt.Errorf("%v gave a response with length > to %v", endpoint, maxLength)
	}
	return result, err
}

func getResponse(url string) (string, error) {
	client := http.Client{
		Transport: httputils.CreateHTTPTransport(),
		Timeout:   time.Duration(config.Datadog.GetInt("gce_metadata_timeout")) * time.Millisecond,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Metadata-Flavor", "Google")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("status code %d trying to GET %s", res.StatusCode, url)
	}

	defer res.Body.Close()
	all, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("GCE hostname, error reading response body: %s", err)
	}

	// Some cloud platforms will respond with an empty body, causing the agent to assume a faulty hostname
	if len(all) <= 0 {
		return "", fmt.Errorf("empty response body")
	}

	return string(all), nil
}

// HostnameProvider GCE implementation of the HostnameProvider
func HostnameProvider() (string, error) {
	return GetHostname()
}
