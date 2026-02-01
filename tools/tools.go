//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/google/wire/cmd/wire"
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/pressly/goose/v3/cmd/goose"
	_ "github.com/rakyll/hey"
	_ "github.com/wadey/gocovmerge"
	_ "github.com/yoheimuta/protolint/cmd/protolint"
	_ "go.uber.org/mock/gomock"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "mvdan.cc/gofumpt"
)
