// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package containeropenapi

//go:generate go tool oapi-codegen --include-operation-ids getContainers,getContainer,getContainerMountUsage,postContainerMountUsageRefresh,getContainerLogs,postContainerStart,postContainerStop,postContainerRestart,postContainerRemove,postContainerBatchActions --generate types --package containeropenapi -o zz_generated.container.go ../../../../../openapi/openapi.yaml
