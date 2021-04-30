// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package config

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	coreConfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/snmp/traps"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// ContainerCollectAll is the name of the docker integration that collect logs from all containers
const ContainerCollectAll = "container_collect_all"

// SnmpTraps is the name of the integration that collects logs from SNMP traps received by the Agent
const SnmpTraps = "snmp_traps"

// logs-intake endpoint prefix.
const (
	tcpEndpointPrefix  = "agent-intake.logs."
	httpEndpointPrefix = "agent-http-intake.logs."
)

// logs-intake endpoints depending on the site and environment.
var logsEndpoints = map[string]int{
	"agent-intake.logs.datadoghq.com": 10516,
	"agent-intake.logs.datadoghq.eu":  443,
	"agent-intake.logs.datad0g.com":   10516,
	"agent-intake.logs.datad0g.eu":    443,
}

// HTTPConnectivity is the status of the HTTP connectivity
type HTTPConnectivity bool

var (
	// HTTPConnectivitySuccess is the status for successful HTTP connectivity
	HTTPConnectivitySuccess HTTPConnectivity = true
	// HTTPConnectivityFailure is the status for failed HTTP connectivity
	HTTPConnectivityFailure HTTPConnectivity = false
)

// ContainerCollectAllSource returns a source to collect all logs from all containers.
func ContainerCollectAllSource() *LogSource {
	if coreConfig.Datadog.GetBool("logs_config.container_collect_all") {
		// source to collect all logs from all containers
		return NewLogSource(ContainerCollectAll, &LogsConfig{
			Type:    DockerType,
			Service: "docker",
			Source:  "docker",
		})
	}
	return nil
}

// SNMPTrapsSource returs a source to forward SNMP traps as logs.
func SNMPTrapsSource() *LogSource {
	if traps.IsEnabled() && traps.IsRunning() {
		// source to forward SNMP traps as logs.
		return NewLogSource(SnmpTraps, &LogsConfig{
			Type:    SnmpTrapsType,
			Service: "snmp",
			Source:  "snmp",
		})
	}
	return nil
}

// GlobalProcessingRules returns the global processing rules to apply to all logs.
func GlobalProcessingRules() ([]*ProcessingRule, error) {
	var rules []*ProcessingRule
	var err error
	raw := coreConfig.Datadog.Get("logs_config.processing_rules")
	if raw == nil {
		return rules, nil
	}
	if s, ok := raw.(string); ok && s != "" {
		err = json.Unmarshal([]byte(s), &rules)
	} else {
		err = coreConfig.Datadog.UnmarshalKey("logs_config.processing_rules", &rules)
	}
	if err != nil {
		return nil, err
	}
	err = ValidateProcessingRules(rules)
	if err != nil {
		return nil, err
	}
	err = CompileProcessingRules(rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// BuildEndpoints returns the endpoints to send logs.
func BuildEndpoints(httpConnectivity HTTPConnectivity) (*Endpoints, error) {
	coreConfig.SanitizeAPIKeyConfig(coreConfig.Datadog, "logs_config.api_key")
	return BuildEndpointsWithConfig(logsConfigDefaultKeys, httpEndpointPrefix, httpConnectivity)
}

// BuildEndpointsWithConfig returns the endpoints to send logs.
func BuildEndpointsWithConfig(logsConfig *LogsConfigKeys, endpointPrefix string, httpConnectivity HTTPConnectivity) (*Endpoints, error) {
	if logsConfig.devModeNoSSL() {
		log.Warnf("Use of illegal configuration parameter, if you need to send your logs to a proxy, "+
			"please use '%s' and '%s' instead", logsConfig.getConfigKey("logs_dd_url"), logsConfig.getConfigKey("logs_no_ssl"))
	}
	if logsConfig.isForceHTTPUse() || (bool(httpConnectivity) && !(logsConfig.isForceTCPUse() || logsConfig.isSocks5ProxySet() || logsConfig.hasAdditionalEndpoints())) {
		return BuildHTTPEndpointsWithConfig(logsConfig, endpointPrefix)
	}
	log.Warnf("You are currently sending Logs to Datadog through TCP (either because %s or %s is set or the HTTP connectivity test has failed) "+
		"To benefit from increased reliability and better network performances, "+
		"we strongly encourage switching over to compressed HTTPS which is now the default protocol.",
		logsConfig.getConfigKey("use_tcp"), logsConfig.getConfigKey("socks5_proxy_address"))
	return buildTCPEndpoints(logsConfig)
}

// ExpectedTagsDuration returns a duration of the time expected tags will be submitted for.
func ExpectedTagsDuration() time.Duration {
	return logsConfigDefaultKeys.expectedTagsDuration()
}

// IsExpectedTagsSet returns boolean showing if expected tags feature is enabled.
func IsExpectedTagsSet() bool {
	return ExpectedTagsDuration() > 0
}

func buildTCPEndpoints(logsConfig *LogsConfigKeys) (*Endpoints, error) {
	useProto := logsConfig.devModeUseProto()
	proxyAddress := logsConfig.socks5ProxyAddress()
	main := Endpoint{
		APIKey:                  logsConfig.getLogsAPIKey(),
		ProxyAddress:            proxyAddress,
		ConnectionResetInterval: logsConfig.connectionResetInterval(),
	}

	if logsDDURL, defined := logsConfig.logsDDURL(); defined {
		// Proxy settings, expect 'logs_config.logs_dd_url' to respect the format '<HOST>:<PORT>'
		// and '<PORT>' to be an integer.
		// By default ssl is enabled ; to disable ssl set 'logs_config.logs_no_ssl' to true.
		host, port, err := parseAddress(logsDDURL)
		if err != nil {
			return nil, fmt.Errorf("could not parse %s: %v", logsConfig.getConfigKey("logs_dd_url"), err)
		}
		main.Host = host
		main.Port = port
		main.UseSSL = !logsConfig.logsNoSSL()
	} else if logsConfig.usePort443() {
		main.Host = logsConfig.ddURL443()
		main.Port = 443
		main.UseSSL = true
	} else {
		// If no proxy is set, we default to 'logs_config.dd_url' if set, or to 'site'.
		// if none of them is set, we default to the US agent endpoint.
		main.Host = coreConfig.GetMainEndpoint(tcpEndpointPrefix, logsConfig.getConfigKey("dd_url"))
		if port, found := logsEndpoints[main.Host]; found {
			main.Port = port
		} else {
			main.Port = logsConfig.ddPort()
		}
		main.UseSSL = !logsConfig.devModeNoSSL()
	}

	additionals := logsConfig.getAdditionalEndpoints()
	for i := 0; i < len(additionals); i++ {
		additionals[i].UseSSL = main.UseSSL
		additionals[i].ProxyAddress = proxyAddress
		additionals[i].APIKey = coreConfig.SanitizeAPIKey(additionals[i].APIKey)
	}
	return NewEndpoints(main, additionals, useProto, false, 0, 0), nil
}

// LogsConfigKeys stores logs configuration keys stored in YAML configuration files
type LogsConfigKeys struct {
	prefix string
	config coreConfig.Config
}

// logsConfigDefaultKeys defines the default YAML keys used to retrieve logs configuration
var logsConfigDefaultKeys = NewLogsConfigKeys("logs_config.", coreConfig.Datadog)

// NewLogsConfigKeys returns a new logs configuration keys set
func NewLogsConfigKeys(configPrefix string, config coreConfig.Config) *LogsConfigKeys {
	return &LogsConfigKeys{prefix: configPrefix, config: config}
}

func (l *LogsConfigKeys) getConfigKey(key string) string {
	return l.prefix + key
}

func isSetAndNotEmpty(config coreConfig.Config, key string) bool {
	return config.IsSet(key) && len(config.GetString(key)) > 0
}

func (l *LogsConfigKeys) isSetAndNotEmpty(key string) bool {
	return isSetAndNotEmpty(l.config, key)
}

func (l *LogsConfigKeys) ddURL() string {
	return l.config.GetString(l.getConfigKey("dd_url"))
}

func (l *LogsConfigKeys) ddURL443() string {
	return l.config.GetString(l.getConfigKey("dd_url_443"))
}

func (l *LogsConfigKeys) logsDDURL() (string, bool) {
	configKey := l.getConfigKey("logs_dd_url")
	return l.config.GetString(configKey), l.isSetAndNotEmpty(configKey)
}

func (l *LogsConfigKeys) ddPort() int {
	return l.config.GetInt(l.getConfigKey("dd_port"))
}

func (l *LogsConfigKeys) isSocks5ProxySet() bool {
	return len(l.socks5ProxyAddress()) > 0
}

func (l *LogsConfigKeys) socks5ProxyAddress() string {
	return l.config.GetString(l.getConfigKey("socks5_proxy_address"))
}

func (l *LogsConfigKeys) isForceTCPUse() bool {
	return l.config.GetBool(l.getConfigKey("use_tcp"))
}

func (l *LogsConfigKeys) usePort443() bool {
	return l.config.GetBool(l.getConfigKey("use_port_443"))
}

func (l *LogsConfigKeys) isForceHTTPUse() bool {
	return l.config.GetBool(l.getConfigKey("use_http"))
}

func (l *LogsConfigKeys) logsNoSSL() bool {
	return l.config.GetBool(l.getConfigKey("logs_no_ssl"))
}

func (l *LogsConfigKeys) devModeNoSSL() bool {
	return l.config.GetBool(l.getConfigKey("dev_mode_no_ssl"))
}

func (l *LogsConfigKeys) devModeUseProto() bool {
	return l.config.GetBool(l.getConfigKey("dev_mode_use_proto"))
}

func (l *LogsConfigKeys) compressionLevel() int {
	return l.config.GetInt(l.getConfigKey("compression_level"))
}

func (l *LogsConfigKeys) useCompression() bool {
	return l.config.GetBool(l.getConfigKey("use_compression"))
}

func (l *LogsConfigKeys) hasAdditionalEndpoints() bool {
	return len(l.getAdditionalEndpoints()) > 0
}

// getLogsAPIKey provides the dd api key used by the main logs agent sender.
func (l *LogsConfigKeys) getLogsAPIKey() string {
	if configKey := l.getConfigKey("api_key"); l.isSetAndNotEmpty(configKey) {
		return coreConfig.SanitizeAPIKey(l.config.GetString(configKey))
	}
	return coreConfig.SanitizeAPIKey(l.config.GetString("api_key"))
}

func (l *LogsConfigKeys) connectionResetInterval() time.Duration {
	return time.Duration(l.config.GetInt(l.getConfigKey("connection_reset_interval"))) * time.Second

}

func (l *LogsConfigKeys) getAdditionalEndpoints() []Endpoint {
	var endpoints []Endpoint
	var err error
	configKey := l.getConfigKey("additional_endpoints")
	raw := l.config.Get(configKey)
	if raw == nil {
		return endpoints
	}
	if s, ok := raw.(string); ok && s != "" {
		err = json.Unmarshal([]byte(s), &endpoints)
	} else {
		err = l.config.UnmarshalKey(configKey, &endpoints)
	}
	if err != nil {
		log.Warnf("Could not parse additional_endpoints for logs: %v", err)
	}
	return endpoints
}

func (l *LogsConfigKeys) expectedTagsDuration() time.Duration {
	return l.config.GetDuration(l.getConfigKey("expected_tags_duration")) * time.Second
}

func (l *LogsConfigKeys) taggerWarmupDuration() time.Duration {
	return l.config.GetDuration(l.getConfigKey("tagger_warmup_duration")) * time.Second
}

func (l *LogsConfigKeys) batchWaitFromKey() time.Duration {
	batchWait := l.config.GetInt(l.getConfigKey("batch_wait"))
	if batchWait < 1 || 10 < batchWait {
		log.Warnf("Invalid batch_wait: %v should be in [1, 10], fallback on %v", batchWait, coreConfig.DefaultBatchWait)
		return coreConfig.DefaultBatchWait * time.Second
	}
	return (time.Duration(batchWait) * time.Second)
}

func (l *LogsConfigKeys) batchMaxConcurrentSend() int {
	batchMaxConcurrentSend := l.config.GetInt(l.getConfigKey("batch_max_concurrent_send"))
	if batchMaxConcurrentSend < 0 {
		log.Warnf("Invalid batch_max_concurrent_send: %v should be >= 0, fallback on %v", batchMaxConcurrentSend, coreConfig.DefaultBatchMaxConcurrentSend)
		return coreConfig.DefaultBatchMaxConcurrentSend
	}
	return batchMaxConcurrentSend
}

// AggregationTimeout is used when performing aggregation operations
func (l *LogsConfigKeys) aggregationTimeout() time.Duration {
	return l.config.GetDuration(l.getConfigKey("aggregation_timeout")) * time.Millisecond
}

// BuildHTTPEndpoints returns the HTTP endpoints to send logs to.
func BuildHTTPEndpoints() (*Endpoints, error) {
	return BuildHTTPEndpointsWithConfig(logsConfigDefaultKeys, httpEndpointPrefix)
}

// BuildHTTPEndpointsWithConfig uses two arguments that instructs it how to access configuration parameters, then returns the HTTP endpoints to send logs to. This function is able to default to the 'classic' BuildHTTPEndpoints() w ldHTTPEndpointsWithConfigdefault variables logsConfigDefaultKeys and httpEndpointPrefix
func BuildHTTPEndpointsWithConfig(logsConfig *LogsConfigKeys, endpointPrefix string) (*Endpoints, error) {
	// Provide default values for legacy settings when the configuration key does not exist
	defaultNoSSL := logsConfig.logsNoSSL()

	main := Endpoint{
		APIKey:                  logsConfig.getLogsAPIKey(),
		UseCompression:          logsConfig.useCompression(),
		CompressionLevel:        logsConfig.compressionLevel(),
		ConnectionResetInterval: logsConfig.connectionResetInterval(),
	}

	if logsDDURL, logsDDURLDefined := logsConfig.logsDDURL(); logsDDURLDefined {
		host, port, err := parseAddress(logsDDURL)
		if err != nil {
			return nil, fmt.Errorf("could not parse %s: %v", logsConfig.getConfigKey("logs_dd_url"), err)
		}
		main.Host = host
		main.Port = port
		main.UseSSL = !defaultNoSSL
	} else {
		main.Host = coreConfig.GetMainEndpoint(endpointPrefix, logsConfig.getConfigKey("dd_url"))
		main.UseSSL = !logsConfig.devModeNoSSL()
	}

	additionals := logsConfig.getAdditionalEndpoints()
	for i := 0; i < len(additionals); i++ {
		additionals[i].UseSSL = main.UseSSL
		additionals[i].APIKey = coreConfig.SanitizeAPIKey(additionals[i].APIKey)
	}

	batchWait := logsConfig.batchWaitFromKey()
	batchMaxConcurrentSend := logsConfig.batchMaxConcurrentSend()

	return NewEndpoints(main, additionals, false, true, batchWait, batchMaxConcurrentSend), nil
}

// parseAddress returns the host and the port of the address.
func parseAddress(address string) (string, int, error) {
	host, portString, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

// TaggerWarmupDuration is used to configure the tag providers
func TaggerWarmupDuration() time.Duration {
	return logsConfigDefaultKeys.taggerWarmupDuration()
}

// AggregationTimeout is used when performing aggregation operations
func AggregationTimeout() time.Duration {
	return logsConfigDefaultKeys.aggregationTimeout()
}
