package main

import (
	"context"

	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
)

const navivoxAndroidPackage = navivoxapp.AndroidPackage

func defaultOpenNavivoxAndroid() bool {
	return navivoxapp.DefaultOpenAndroid()
}

func navivoxAndroidEnvironment() bool {
	return navivoxapp.AndroidEnvironment()
}

func openNavivoxAndroid(ctx context.Context, descriptor, pkg string) error {
	return navivoxapp.OpenAndroid(ctx, descriptor, pkg)
}

func navivoxDescriptorSharePayload(descriptor string) string {
	return navivoxapp.SharePayload(descriptor)
}

func redactNavivoxDescriptor(text string) string {
	return navivoxapp.Redact(text)
}

func shouldOpenNavivoxAndroid(open, noOpen bool) bool {
	return navivoxapp.ShouldOpenAndroid(open, noOpen)
}
