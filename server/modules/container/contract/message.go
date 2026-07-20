package contract

// MessageKey identifies a stable container management message key.
type MessageKey string

// String returns the canonical message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// ContainerMenuTitle identifies the Docker provider navigation title.
	ContainerMenuTitle MessageKey = "menu.docker.title"
	// ContainerListMenuTitle identifies the shortened container-list navigation title.
	ContainerListMenuTitle MessageKey = "menu.container.title"
	// DockerImageMenuTitle identifies the image-management navigation title.
	DockerImageMenuTitle MessageKey = "menu.docker.image.title"
	// DockerVolumeMenuTitle identifies the Docker volume navigation title.
	DockerVolumeMenuTitle MessageKey = "menu.dockerVolume.title"
	// DockerNetworkMenuTitle identifies the Docker network navigation title.
	DockerNetworkMenuTitle MessageKey = "menu.docker.network.title"
	// ContainerMenuSectionTitle identifies the visual-only runtime sidebar section title.
	ContainerMenuSectionTitle MessageKey = "menu.section.runtime"
	// ContainerRuntimeDisabled identifies disabled runtime errors.
	ContainerRuntimeDisabled MessageKey = "ops.container.error.runtimeDisabled"
	// ContainerRuntimeSocketMissing identifies missing runtime socket errors.
	ContainerRuntimeSocketMissing MessageKey = "ops.container.error.runtimeSocketMissing"
	// ContainerRuntimePermissionDenied identifies runtime socket permission errors.
	ContainerRuntimePermissionDenied MessageKey = "ops.container.error.runtimePermissionDenied"
	// ContainerRuntimeUnavailable identifies unavailable runtime connection errors.
	ContainerRuntimeUnavailable MessageKey = "ops.container.error.runtimeUnavailable"
	// ContainerNotFound identifies missing container errors.
	ContainerNotFound MessageKey = "ops.container.error.containerNotFound"
	// DockerVolumeNotFound identifies missing Docker volume errors.
	DockerVolumeNotFound MessageKey = "ops.container.error.volumeNotFound"
	// DockerVolumeConflict identifies Docker volume removal conflicts.
	DockerVolumeConflict MessageKey = "ops.container.error.volumeConflict"
	// ContainerMountNotFound identifies missing container mount errors.
	ContainerMountNotFound MessageKey = "ops.container.error.containerMountNotFound"
	// ContainerInvalidRef identifies invalid container reference errors.
	ContainerInvalidRef MessageKey = "ops.container.error.invalidContainerRef"
	// ContainerInvalidListQuery identifies invalid list query parameter errors.
	ContainerInvalidListQuery MessageKey = "ops.container.error.invalidListQuery"
	// ContainerInvalidBatchAction identifies invalid batch action request errors.
	ContainerInvalidBatchAction MessageKey = "ops.container.error.invalidBatchAction"
	// DockerNetworkInvalidRequest identifies malformed Docker network commands.
	DockerNetworkInvalidRequest MessageKey = "ops.docker.network.error.invalidRequest"
	// DockerNetworkNotFound identifies a missing Docker network.
	DockerNetworkNotFound MessageKey = "ops.docker.network.error.notFound"
	// DockerNetworkConfirmationMismatch identifies a failed destructive-action confirmation.
	DockerNetworkConfirmationMismatch MessageKey = "ops.docker.network.error.confirmationMismatch"
	// DockerNetworkDefaultProtected identifies a protected Docker default network.
	DockerNetworkDefaultProtected MessageKey = "ops.docker.network.error.defaultProtected"
	// DockerNetworkInUse identifies a network with attached containers.
	DockerNetworkInUse MessageKey = "ops.docker.network.error.inUse"
	// DockerNetworkCreateCompleted identifies a completed Docker network creation.
	DockerNetworkCreateCompleted MessageKey = "ops.docker.network.action.create.completed"
	// DockerNetworkRemoveCompleted identifies a completed Docker network removal.
	DockerNetworkRemoveCompleted MessageKey = "ops.docker.network.action.remove.completed"
	// ContainerInvalidState identifies invalid action state errors.
	ContainerInvalidState MessageKey = "ops.container.error.invalidState"
	// ContainerEventsUnavailable identifies runtime event history read failures.
	ContainerEventsUnavailable MessageKey = "ops.container.error.eventsUnavailable"
	// ContainerLogsTooLarge identifies log limit errors.
	ContainerLogsTooLarge MessageKey = "ops.container.error.logsTooLarge"
	// ContainerInvalidLogQuery identifies invalid log query parameter errors.
	ContainerInvalidLogQuery MessageKey = "ops.container.error.invalidLogQuery"
	// ContainerShellDisabled identifies disabled shell feature errors.
	ContainerShellDisabled MessageKey = "ops.container.error.shellDisabled"
	// ContainerShellForbidden identifies shell permission errors.
	ContainerShellForbidden MessageKey = "ops.container.error.shellForbidden"
	// ContainerShellTicketInvalid identifies invalid shell ticket errors.
	ContainerShellTicketInvalid MessageKey = "ops.container.error.shellTicketInvalid"
	// ContainerShellTicketExpired identifies expired shell ticket errors.
	ContainerShellTicketExpired MessageKey = "ops.container.error.shellTicketExpired"
	// ContainerShellTicketUsed identifies consumed shell ticket errors.
	ContainerShellTicketUsed MessageKey = "ops.container.error.shellTicketUsed"
	// ContainerShellOriginDenied identifies denied shell websocket origin errors.
	ContainerShellOriginDenied MessageKey = "ops.container.error.shellOriginDenied"
	// ContainerShellContainerNotRunning identifies non-running container shell errors.
	ContainerShellContainerNotRunning MessageKey = "ops.container.error.shellContainerNotRunning"
	// ContainerShellCommandNotFound identifies missing shell command errors.
	ContainerShellCommandNotFound MessageKey = "ops.container.error.shellCommandNotFound"
	// ContainerShellInvalidSize identifies invalid shell terminal dimension errors.
	ContainerShellInvalidSize MessageKey = "ops.container.error.shellInvalidSize"
	// ContainerShellSessionFailed identifies generic shell session failures.
	ContainerShellSessionFailed MessageKey = "ops.container.error.shellSessionFailed"
	// ContainerShellUnsupportedControlMessage identifies unsupported terminal control payload errors.
	ContainerShellUnsupportedControlMessage MessageKey = "ops.container.error.shellUnsupportedControlMessage"
	// ContainerTimeout identifies runtime timeout errors.
	ContainerTimeout MessageKey = "ops.container.error.timeout"
	// ContainerMountUsageUnsupported identifies unsupported mount usage errors.
	ContainerMountUsageUnsupported MessageKey = "ops.container.error.mountUsageUnsupported"
	// ContainerDangerousActionsDisabled identifies disabled action errors.
	ContainerDangerousActionsDisabled MessageKey = "ops.container.error.dangerousActionsDisabled"
	// DockerImageInvalidReference identifies invalid image pull or target tag input.
	DockerImageInvalidReference MessageKey = "ops.container.error.invalidImageReference"
	// DockerImageInUse identifies image deletion blocked by container references.
	DockerImageInUse MessageKey = "ops.container.error.imageInUse"
	// DockerImageReferencedByMultipleTags 标识因多个 Repository:Tag 引用被拒绝的镜像删除。
	DockerImageReferencedByMultipleTags MessageKey = "ops.container.error.imageReferencedByMultipleTags"
	// DockerImageNotFound 标识删除时 Docker daemon 未找到指定镜像。
	DockerImageNotFound MessageKey = "ops.container.error.imageNotFound"
	// DockerImageCommunicationError 标识 Docker daemon 通信未完成。
	DockerImageCommunicationError MessageKey = "ops.container.error.dockerCommunication"
	// DockerImagePullFailed identifies a Docker daemon pull failure without exposing daemon diagnostics.
	DockerImagePullFailed MessageKey = "ops.container.error.imagePullFailed"
	// DockerImageTagFailed identifies a Docker daemon tag failure without exposing daemon diagnostics.
	DockerImageTagFailed MessageKey = "ops.container.error.imageTagFailed"
	// DockerImageRemoveFailed identifies a Docker daemon remove failure without exposing daemon diagnostics.
	DockerImageRemoveFailed MessageKey = "ops.container.error.imageRemoveFailed"
	// DockerImageTagNotAssociated 表示标签不引用请求的镜像。
	DockerImageTagNotAssociated MessageKey = "ops.container.error.imageTagNotAssociated"
	// DockerImagePullCompleted identifies successful image pull completion.
	DockerImagePullCompleted MessageKey = "ops.container.image.pull.completed"
	// DockerImageTagCompleted identifies successful image tag completion.
	DockerImageTagCompleted MessageKey = "ops.container.image.tag.completed"
	// DockerImageUntagCompleted 表示镜像标签移除已完成。
	DockerImageUntagCompleted MessageKey = "ops.container.image.untag.completed"
	// DockerImageRemoveCompleted identifies successful image remove completion.
	DockerImageRemoveCompleted MessageKey = "ops.container.image.remove.completed"
	// ContainerAuditShellSessionRequested identifies shell session request audit messages.
	ContainerAuditShellSessionRequested MessageKey = "ops.container.audit.shellSessionRequested"
	// ContainerAuditShellTicketIssued identifies shell ticket issue audit messages.
	ContainerAuditShellTicketIssued MessageKey = "ops.container.audit.shellTicketIssued"
	// ContainerAuditShellTicketRejected identifies shell ticket rejection audit messages.
	ContainerAuditShellTicketRejected MessageKey = "ops.container.audit.shellTicketRejected"
	// ContainerAuditShellSessionStarted identifies shell session start audit messages.
	ContainerAuditShellSessionStarted MessageKey = "ops.container.audit.shellSessionStarted"
	// ContainerAuditShellSessionClosed identifies shell session close audit messages.
	ContainerAuditShellSessionClosed MessageKey = "ops.container.audit.shellSessionClosed"
	// ContainerAuditShellSessionFailed identifies shell session failure audit messages.
	ContainerAuditShellSessionFailed MessageKey = "ops.container.audit.shellSessionFailed"
	// ContainerActionStartCompleted identifies successful start action responses.
	ContainerActionStartCompleted MessageKey = "ops.container.action.start.completed"
	// ContainerActionStopCompleted identifies successful stop action responses.
	ContainerActionStopCompleted MessageKey = "ops.container.action.stop.completed"
	// ContainerActionRestartCompleted identifies successful restart action responses.
	ContainerActionRestartCompleted MessageKey = "ops.container.action.restart.completed"
	// ContainerActionRemoveCompleted identifies successful remove action responses.
	ContainerActionRemoveCompleted MessageKey = "ops.container.action.remove.completed"
	// ContainerBatchActionCompleted identifies fully successful batch action responses.
	ContainerBatchActionCompleted MessageKey = "ops.container.action.batch.completed"
	// ContainerBatchActionPartial identifies partially successful batch action responses.
	ContainerBatchActionPartial MessageKey = "ops.container.action.batch.partial"
	// ContainerBatchActionFailed identifies fully failed batch action responses.
	ContainerBatchActionFailed MessageKey = "ops.container.action.batch.failed"
)
