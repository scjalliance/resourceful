package main

import "embed"

//go:embed www/*
var webfiles embed.FS
