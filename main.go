package main

import (
	"github.com/rizaldntr/storage-service-website-action/config"
	"github.com/rizaldntr/storage-service-website-action/core"
)

func main() {
	cfg := config.Get()
	core.Process(cfg)
}
