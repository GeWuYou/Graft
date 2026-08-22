package build

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func normalizePlatformDigest(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if digestPattern.MatchString(value) {
		return value, true
	}
	if index := strings.LastIndex(value, "@sha256:"); index >= 0 {
		candidate := value[index+1:]
		if digestPattern.MatchString(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func temporaryPlatformTag(reference, legID string) string {
	reference = strings.TrimSpace(reference)
	legID = strings.NewReplacer("/", "-", ":", "-", "@", "-", "_", "-").Replace(strings.TrimSpace(legID))
	return reference + "-graft-" + legID
}

func settleV2Artifact(ctx context.Context, repository buildstore.Repository, taskID uint64, plan moduleapi.BuildExecutionPlan, result moduleapi.BuildArtifactResult, authExecution moduleapi.RegistryAuthExecution) error {
	settler, ok := repository.(buildstore.V2ArtifactSettlementRepository)
	if !ok {
		return errors.New("v2 artifact settlement is unavailable")
	}
	return settler.SettleV2Artifact(ctx, taskID, plan, result, authExecution)
}
