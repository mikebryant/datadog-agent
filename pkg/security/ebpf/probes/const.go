// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build linux

package probes

const (
	// SecurityAgentUID is the UID used for all the runtime security module probes
	SecurityAgentUID = "security"
)

const (
	// DentryResolverKernKey is the key to the kernel dentry resolver tail call program
	DentryResolverKernKey uint32 = iota
	// DentryResolverERPCKey is the key to the eRPC dentry resolver tail call program
	DentryResolverERPCKey
)

const (
	// DentryResolverOpenCallbackKey is the key to the callback program to execute after resolving the dentry of an open event
	DentryResolverOpenCallbackKey uint32 = iota
	// DentryResolverSetAttrCallbackKey is the key to the callback program to execute after resolving the dentry of an setattr event
	DentryResolverSetAttrCallbackKey
	// DentryResolverMkdirCallbackKey is the key to the callback program to execute after resolving the dentry of an mkdir event
	DentryResolverMkdirCallbackKey
	// DentryResolverMountCallbackKey is the key to the callback program to execute after resolving the dentry of an mount event
	DentryResolverMountCallbackKey
	// DentryResolverSecurityInodeRmdirCallbackKey is the key to the callback program to execute after resolving the dentry of an rmdir or unlink event
	DentryResolverSecurityInodeRmdirCallbackKey
	// DentryResolverSetXAttrCallbackKey is the key to the callback program to execute after resolving the dentry of an setxattr event
	DentryResolverSetXAttrCallbackKey
	// DentryResolverUnlinkCallbackKey is the key to the callback program to execute after resolving the dentry of an unlink event
	DentryResolverUnlinkCallbackKey
	// DentryResolverLinkSrcCallbackKey is the key to the callback program to execute after resolving the source dentry of a link event
	DentryResolverLinkSrcCallbackKey
	// DentryResolverLinkDstCallbackKey is the key to the callback program to execute after resolving the destination dentry of a link event
	DentryResolverLinkDstCallbackKey
	// DentryResolverRenameCallbackKey is the key to the callback program to execute after resolving the destination dentry of a rename event
	DentryResolverRenameCallbackKey
)
