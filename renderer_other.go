//go:build !linux

package main

// configureRenderer: the system web views of macOS and Windows always composite on the GPU.
func configureRenderer() (blur bool) { return true }
