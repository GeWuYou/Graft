<template>
  <div
    ref="workspaceRootRef"
    class="project-configuration-workspace"
    :class="{ 'project-configuration-workspace--fullscreen': workspaceFullscreen }"
    data-page-type="editor"
  >
    <management-page-content>
      <management-page-header compact :title="pageHeaderTitle" :description="workspaceCopy.summaryDescription">
        <template #meta>
          <t-space break-line size="small">
            <t-tag :theme="runtimeTheme" variant="light-outline">
              {{ runtimeLabel }}
            </t-tag>
            <t-tag :theme="driftTheme" variant="light-outline">
              {{ driftLabel }}
            </t-tag>
            <t-tag theme="default" variant="light-outline">
              {{ detailRecord?.ownership_mode || '-' }}
            </t-tag>
          </t-space>
        </template>
      </management-page-header>

      <t-loading :loading="workspaceLoading" size="small">
        <template v-if="workspaceReady">
          <section class="project-configuration-workspace__summary-strip">
            <t-card bordered>
              <template #header>
                <div class="project-configuration-workspace__section-head">
                  <div>
                    <h2>{{ workspaceCopy.summaryTitle }}</h2>
                    <p>{{ t('project.detail.configuration.title') }}</p>
                  </div>
                  <t-space size="small" break-line>
                    <t-button theme="default" variant="outline" @click="openSnapshotDrawer">
                      {{ workspaceCopy.snapshotAction }}
                    </t-button>
                  </t-space>
                </div>
              </template>

              <t-descriptions bordered size="small" :column="5">
                <t-descriptions-item :label="t('project.detail.configuration.ownershipMode')">
                  {{ detailRecord?.ownership_mode || '-' }}
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryWorkingDirectoryLabel">
                  <code>{{ detailRecord?.working_directory || '-' }}</code>
                </t-descriptions-item>
                <t-descriptions-item :label="t('project.detail.configuration.driftStatus')">
                  {{ driftLabel }}
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryCurrentPathLabel">
                  <code>{{ currentWorkspacePathLabel }}</code>
                </t-descriptions-item>
                <t-descriptions-item :label="workspaceCopy.summaryOpenTabsLabel">
                  {{ openTabs.length }}
                </t-descriptions-item>
              </t-descriptions>
            </t-card>
          </section>

          <section ref="workspaceShellRef" class="project-configuration-workspace__main-grid">
            <aside class="project-configuration-workspace__sidebar" :style="sidebarPaneStyle">
              <t-card class="project-configuration-workspace__browser-card" bordered>
                <template #header>
                  <div class="project-configuration-workspace__section-head">
                    <h2>{{ workspaceCopy.fileTreeTitle }}</h2>
                    <t-tooltip
                      :content="showHiddenFiles ? workspaceCopy.hideHiddenAction : workspaceCopy.showHiddenAction"
                      placement="bottom"
                      theme="light"
                    >
                      <t-button
                        class="project-configuration-workspace__tree-toolbar-button"
                        theme="default"
                        variant="text"
                        shape="square"
                        size="small"
                        :data-testid="showHiddenFiles ? 'workspace-hide-hidden-toggle' : 'workspace-show-hidden-toggle'"
                        @click="showHiddenFiles = !showHiddenFiles"
                      >
                        <template #icon>
                          <browse-off-icon v-if="showHiddenFiles" />
                          <browse-icon v-else />
                        </template>
                      </t-button>
                    </t-tooltip>
                  </div>
                </template>

                <t-alert
                  v-if="browserError"
                  class="project-configuration-workspace__browser-alert"
                  theme="error"
                  :message="browserError"
                />

                <t-loading :loading="browserLoading" size="small">
                  <div v-if="workspaceFlatRows.length" class="project-configuration-workspace__tree graft-scrollbar">
                    <template v-for="row in workspaceFlatRows" :key="row.item.relative_path || row.item.name">
                      <div
                        class="project-configuration-workspace__tree-row"
                        :class="{
                          'project-configuration-workspace__tree-row--active': isWorkspaceItemActive(row.item),
                          'project-configuration-workspace__tree-row--readonly':
                            row.item.node_type === 'file' && (!row.item.readable || !row.item.editable),
                        }"
                        :style="{ '--workspace-tree-depth': String(row.depth) }"
                      >
                        <button
                          v-if="row.item.node_type === 'directory'"
                          class="project-configuration-workspace__tree-expander"
                          type="button"
                          :aria-expanded="row.expanded"
                          @click.stop="toggleWorkspaceDirectory(row.item)"
                        >
                          <span class="project-configuration-workspace__tree-expander-icon">
                            {{ row.expanded ? '▾' : '▸' }}
                          </span>
                        </button>
                        <span v-else class="project-configuration-workspace__tree-expander-placeholder" />

                        <button
                          class="project-configuration-workspace__tree-entry"
                          :data-testid="workspaceEntryTestId(row.item)"
                          type="button"
                          @click="handleWorkspaceEntry(row.item)"
                        >
                          <span class="project-configuration-workspace__browser-icon" aria-hidden="true">
                            <folder-icon v-if="row.item.node_type === 'directory'" />
                            <span
                              v-else-if="row.item.file_kind === 'compose'"
                              class="project-configuration-workspace__docker-icon"
                            >
                              <svg viewBox="0 0 24 24" role="presentation">
                                <path
                                  d="M9.3 7.2h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm-5.4 3h2.3v2.1H6.6zm2.7 0h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm2.7 0h2.3v2.1h-2.3zm-1.2 3.2c.9 0 1.7-.2 2.4-.6.4.8 1.1 1.4 2 1.7 1.7.7 3.7.2 5-1.3-1-.4-1.7-1.4-1.7-2.6 0-1.2.7-2.2 1.7-2.6-.5-.7-1.4-1.2-2.4-1.2-.6 0-1.2.2-1.7.5-.6-1.4-1.9-2.4-3.5-2.6l-.8 1.3.7 1.1c-.2 0-.5-.1-.7-.1H5.4v4.4c0 1.2.5 2.4 1.4 3.2 1 .9 2.3 1.4 3.7 1.4h2z"
                                  fill="currentColor"
                                />
                              </svg>
                            </span>
                            <command-icon v-else-if="row.item.file_kind === 'env'" />
                            <file-code-icon
                              v-else-if="row.item.file_kind === 'config' || row.item.file_kind === 'text'"
                            />
                            <file-icon v-else />
                          </span>
                          <t-tooltip
                            v-if="workspaceItemTooltip(row.item)"
                            :content="workspaceItemTooltip(row.item)"
                            placement="top-left"
                            theme="light"
                          >
                            <span class="project-configuration-workspace__browser-main">
                              <span class="project-configuration-workspace__browser-title">{{ row.item.name }}</span>
                            </span>
                          </t-tooltip>
                          <span v-else class="project-configuration-workspace__browser-main">
                            <span class="project-configuration-workspace__browser-title">{{ row.item.name }}</span>
                          </span>
                        </button>

                        <div
                          class="project-configuration-workspace__tree-actions"
                          :class="{
                            'project-configuration-workspace__tree-actions--visible': Boolean(row.item.project_note),
                          }"
                        >
                          <t-tooltip :content="workspaceCopy.annotationAction" theme="light">
                            <t-button
                              class="project-configuration-workspace__annotation-button"
                              theme="default"
                              variant="text"
                              shape="square"
                              size="small"
                              tag="div"
                              :data-testid="workspaceAnnotationTestId(row.item)"
                              @click.stop="handleWorkspaceAnnotation(row.item)"
                            >
                              <template #icon>
                                <edit-1-icon />
                              </template>
                            </t-button>
                          </t-tooltip>
                        </div>
                      </div>
                      <p
                        v-if="row.error && row.expanded"
                        class="project-configuration-workspace__tree-error"
                        :style="{ '--workspace-tree-depth': String(row.depth + 1) }"
                      >
                        {{ row.error }}
                      </p>
                    </template>
                  </div>
                  <t-empty v-else :description="workspaceCopy.filesEmpty" />
                </t-loading>
              </t-card>
            </aside>

            <div
              v-if="isSidebarResizable"
              class="project-configuration-workspace__splitter"
              role="separator"
              :aria-label="workspaceCopy.resizeFileTreeAriaLabel"
              aria-orientation="vertical"
              tabindex="0"
              @pointerdown.prevent="startSidebarResize"
            >
              <span class="project-configuration-workspace__splitter-grip" />
            </div>

            <div ref="editorStackHostRef" class="project-configuration-workspace__editor-stack">
              <content-viewer-frame
                :default-height="editorFrameHeight"
                :exit-fullscreen-label="workspaceCopy.exitFullscreenAction"
                :fill-height="workspaceFullscreen"
                :fullscreen-label="workspaceCopy.fullscreenAction"
                fullscreen-surface-padding="none"
                resize-handle-label="Resize Editor Height"
                :resizable="!workspaceFullscreen"
                :show-fullscreen-button="false"
                :storage-key="EDITOR_HEIGHT_STORAGE_KEY"
                surface-padding="none"
              >
                <template #header-actions>
                  <t-space size="small" break-line>
                    <t-button theme="default" variant="outline" :disabled="!activeBuffer" @click="reloadActiveFile">
                      {{ workspaceCopy.reloadAction }}
                    </t-button>
                    <t-button theme="default" variant="outline" @click="workspaceFullscreen = !workspaceFullscreen">
                      {{ workspaceFullscreen ? workspaceCopy.exitFullscreenAction : workspaceCopy.fullscreenAction }}
                    </t-button>
                    <t-button
                      theme="default"
                      variant="outline"
                      :loading="Boolean(activeBuffer?.saving) || diffLoading"
                      :disabled="!canSaveActiveBuffer"
                      @click="saveActiveFile"
                    >
                      {{ workspaceCopy.saveAction }}
                    </t-button>
                    <t-button theme="default" variant="outline" :loading="validateLoading" @click="runProjectValidate">
                      {{ workspaceCopy.validateAction }}
                    </t-button>
                    <t-button theme="primary" :loading="deployLoading" @click="runProjectDeploy">
                      {{ workspaceCopy.deployAction }}
                    </t-button>
                  </t-space>
                </template>

                <div class="project-configuration-workspace__editor-surface">
                  <t-tabs
                    v-if="openTabBuffers.length"
                    v-model:value="activeTabPath"
                    class="project-configuration-workspace__tabs"
                    theme="card"
                  >
                    <t-tab-panel
                      v-for="tab in openTabBuffers"
                      :key="tab.path"
                      :value="tab.path"
                      removable
                      @remove="handleCloseTab(tab.path)"
                    >
                      <template #label>
                        <t-dropdown
                          trigger="context-menu"
                          :hide-after-item-click="true"
                          :min-column-width="128"
                          :popup-props="{
                            onVisibleChange: (visible: boolean, ctx: PopupVisibleChangeContext) =>
                              handleFileTabMenuClick(visible, ctx, tab.path),
                            visible: activeFileTabPathForMenu === tab.path,
                          }"
                          :data-testid="workspaceFileTabMenuTestId(tab.path)"
                        >
                          <span class="project-configuration-workspace__tab-label">
                            <span v-if="isFileDirty(tab.path)" class="project-configuration-workspace__tab-dirty"
                              >●</span
                            >
                            <span>{{ tab.name }}</span>
                          </span>
                          <template #dropdown>
                            <t-dropdown-menu>
                              <t-dropdown-item
                                :data-testid="workspaceFileTabMenuItemTestId(tab.path, 'refresh')"
                                @click="() => handleRefreshFileTab(tab.path)"
                              >
                                {{ t('layout.tagTabs.refresh') }}
                              </t-dropdown-item>
                              <t-dropdown-item
                                :data-testid="workspaceFileTabMenuItemTestId(tab.path, 'close-left')"
                                :disabled="!hasClosableFileTabsAhead(tab.path)"
                                @click="() => handleCloseFileTabsAhead(tab.path)"
                              >
                                {{ t('layout.tagTabs.closeLeft') }}
                              </t-dropdown-item>
                              <t-dropdown-item
                                :data-testid="workspaceFileTabMenuItemTestId(tab.path, 'close-right')"
                                :disabled="!hasClosableFileTabsBehind(tab.path)"
                                @click="() => handleCloseFileTabsBehind(tab.path)"
                              >
                                {{ t('layout.tagTabs.closeRight') }}
                              </t-dropdown-item>
                              <t-dropdown-item
                                :data-testid="workspaceFileTabMenuItemTestId(tab.path, 'close-other')"
                                :disabled="!hasClosableOtherFileTabs(tab.path)"
                                @click="() => handleCloseOtherFileTabs(tab.path)"
                              >
                                {{ t('layout.tagTabs.closeOther') }}
                              </t-dropdown-item>
                              <t-dropdown-item
                                :data-testid="workspaceFileTabMenuItemTestId(tab.path, 'close-all')"
                                :disabled="!hasClosableFileTabs"
                                @click="handleCloseAllFileTabs"
                              >
                                {{ t('layout.tagTabs.closeAll') }}
                              </t-dropdown-item>
                            </t-dropdown-menu>
                          </template>
                        </t-dropdown>
                      </template>
                    </t-tab-panel>
                  </t-tabs>

                  <t-alert
                    v-if="activeBuffer && !activeBuffer.editable"
                    class="project-configuration-workspace__editor-alert"
                    theme="warning"
                    :message="workspaceCopy.readonlyHint"
                  />
                  <t-alert
                    v-if="activeBuffer?.fileKind === 'env'"
                    class="project-configuration-workspace__editor-alert"
                    theme="info"
                    :message="workspaceCopy.envRedeployHint"
                  />
                  <t-alert
                    v-if="activeBuffer?.error"
                    class="project-configuration-workspace__editor-alert"
                    theme="error"
                    :message="activeBuffer.error"
                  />

                  <div v-if="activeBuffer" class="project-configuration-workspace__editor-stage">
                    <t-loading
                      class="project-configuration-workspace__editor-loading"
                      :loading="activeBuffer.loading"
                      size="small"
                    >
                      <project-monaco-surface
                        v-if="!activeBuffer.error"
                        v-model="activeBuffer.content"
                        class="project-configuration-workspace__monaco-editor"
                        :editor-aria-label="workspaceCopy.editorAriaLabel"
                        :language="activeBuffer.language"
                        :model-key="activeBuffer.path"
                        :options="editorOptions"
                        :read-only="!activeBuffer.editable"
                        test-id="workspace-monaco-editor"
                      />
                    </t-loading>
                  </div>
                  <t-empty v-else :description="workspaceCopy.tabsEmpty" />
                </div>
              </content-viewer-frame>
            </div>
          </section>
        </template>

        <t-empty v-else-if="!workspaceError" :description="t('project.list.retry')" />
      </t-loading>
    </management-page-content>

    <t-drawer v-model:visible="snapshotDrawerVisible" :header="workspaceCopy.snapshotDrawerTitle" size="720px">
      <t-loading :loading="snapshotLoading" size="small">
        <template v-if="snapshotPreview">
          <t-descriptions bordered size="small" :column="1">
            <t-descriptions-item :label="t('project.detail.configuration.previewHash')">
              <t-tooltip :content="snapshotPreview.config_hash" placement="top-left" theme="light">
                <code class="project-configuration-workspace__hash-text">
                  {{ formatWorkspaceHash(snapshotPreview.config_hash) }}
                </code>
              </t-tooltip>
            </t-descriptions-item>
            <t-descriptions-item :label="t('project.detail.configuration.previewUpdatedAt')">
              {{ formatProjectTime(locale, snapshotPreview.refreshed_at) }}
            </t-descriptions-item>
          </t-descriptions>
          <div class="project-configuration-workspace__drawer-viewer">
            <project-monaco-surface
              class="project-configuration-workspace__monaco-viewer"
              :model-value="snapshotPreview.normalized_compose_yaml"
              :editor-aria-label="workspaceCopy.snapshotViewerAriaLabel"
              language="yaml"
              model-key="snapshot-preview"
              :options="readonlyOptions"
              read-only
              test-id="snapshot-monaco-viewer"
            />
          </div>
        </template>
        <t-empty v-else :description="t('project.detail.configuration.previewEmpty')" />
      </t-loading>
    </t-drawer>

    <t-dialog
      v-model:visible="resultDialogVisible"
      :dialog-class-name="resultDialogClassName"
      :dialog-style="resultDialogStyle"
      :close-btn="!resultDialogFullscreen"
      :footer="false"
      :header="false"
      :mode="resultDialogFullscreen ? 'full-screen' : 'modal'"
      placement="center"
      :close-on-overlay-click="false"
      :close-on-esc-keydown="true"
      destroy-on-close
      :on-opened="handleResultDialogOpened"
      :top="resultDialogTop"
      :width="resultDialogWidth"
    >
      <div class="project-configuration-workspace__result-dialog">
        <div class="project-configuration-workspace__result-dialog-header">
          <div class="project-configuration-workspace__section-head">
            <div>
              <h2>
                {{
                  resultDialogMode === 'diff'
                    ? t('project.detail.configuration.diffTitle')
                    : t('project.detail.configuration.validationTitle')
                }}
              </h2>
              <p v-if="resultDialogMode === 'diff'">{{ workspaceCopy.diffConfirmBody }}</p>
            </div>
            <t-space size="small">
              <t-button
                theme="default"
                variant="outline"
                size="small"
                data-testid="configuration-result-fullscreen-toggle"
                @click="toggleResultDialogFullscreen"
              >
                {{ resultDialogFullscreen ? workspaceCopy.exitFullscreenAction : workspaceCopy.fullscreenAction }}
              </t-button>
            </t-space>
          </div>
        </div>

        <div
          v-if="resultDialogMode === 'diff'"
          class="project-configuration-workspace__diff-surface"
          data-testid="configuration-diff-modal"
        >
          <div class="project-configuration-workspace__diff-sidebar graft-scrollbar">
            <div class="project-configuration-workspace__diff-sidebar-head">
              <span class="project-configuration-workspace__diff-sidebar-title">{{ workspaceCopy.diffTreeTitle }}</span>
              <span class="project-configuration-workspace__diff-sidebar-caption">
                {{ workspaceCopy.workspaceRootLabel }}
              </span>
            </div>
            <div
              v-for="row in diffTreeRows"
              :key="row.path"
              class="project-configuration-workspace__tree-row project-configuration-workspace__tree-row--diff"
              :class="{
                'project-configuration-workspace__tree-row--active':
                  row.type === 'file' && selectedDiffFile?.path === row.path,
              }"
              :style="{ '--workspace-tree-depth': String(row.depth) }"
            >
              <span class="project-configuration-workspace__tree-expander-placeholder" />
              <button
                class="project-configuration-workspace__tree-entry"
                :data-testid="diffFileTestId(row.path)"
                type="button"
                :disabled="row.type !== 'file'"
                @click="row.type === 'file' && (selectedDiffFilePath = row.path)"
              >
                <span class="project-configuration-workspace__browser-icon" aria-hidden="true">
                  <folder-icon v-if="row.type === 'directory'" />
                  <span v-else-if="row.file?.kind === 'compose'" class="project-configuration-workspace__docker-icon">
                    <svg viewBox="0 0 24 24" role="presentation">
                      <path
                        d="M9.3 7.2h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm-5.4 3h2.3v2.1H6.6zm2.7 0h2.3v2.1H9.3zm2.7 0h2.3v2.1H12zm2.7 0h2.3v2.1h-2.3zm-1.2 3.2c.9 0 1.7-.2 2.4-.6.4.8 1.1 1.4 2 1.7 1.7.7 3.7.2 5-1.3-1-.4-1.7-1.4-1.7-2.6 0-1.2.7-2.2 1.7-2.6-.5-.7-1.4-1.2-2.4-1.2-.6 0-1.2.2-1.7.5-.6-1.4-1.9-2.4-3.5-2.6l-.8 1.3.7 1.1c-.2 0-.5-.1-.7-.1H5.4v4.4c0 1.2.5 2.4 1.4 3.2 1 .9 2.3 1.4 3.7 1.4h2z"
                        fill="currentColor"
                      />
                    </svg>
                  </span>
                  <command-icon v-else-if="row.file?.kind === 'env'" />
                  <file-code-icon v-else />
                </span>
                <span class="project-configuration-workspace__browser-main">
                  <span class="project-configuration-workspace__browser-title">{{ row.name }}</span>
                </span>
              </button>
            </div>
          </div>

          <div class="project-configuration-workspace__diff-stage">
            <t-alert theme="warning" :message="t('project.detail.configuration.diffHasChanges')" />
            <div v-if="diffResult?.warnings?.length" class="project-configuration-workspace__warning-list">
              <t-alert v-for="warning in diffResult.warnings" :key="warning" theme="warning" :message="warning" />
            </div>
            <div v-if="selectedDiffFile" class="project-configuration-workspace__diff-pane-heads">
              <div class="project-configuration-workspace__diff-pane-head">
                <span class="project-configuration-workspace__diff-pane-label">
                  {{ t('project.detail.configuration.currentHash') }}
                </span>
                <t-tooltip :content="selectedDiffFile.current_hash" placement="top-left" theme="light">
                  <code
                    class="project-configuration-workspace__hash-text"
                    data-testid="configuration-diff-current-hash"
                  >
                    {{ formatWorkspaceHash(selectedDiffFile.current_hash) }}
                  </code>
                </t-tooltip>
              </div>
              <div class="project-configuration-workspace__diff-pane-head">
                <span class="project-configuration-workspace__diff-pane-label">
                  {{ t('project.detail.configuration.proposedHash') }}
                </span>
                <t-tooltip :content="selectedDiffFile.proposed_hash" placement="top-left" theme="light">
                  <code
                    class="project-configuration-workspace__hash-text"
                    data-testid="configuration-diff-proposed-hash"
                  >
                    {{ formatWorkspaceHash(selectedDiffFile.proposed_hash) }}
                  </code>
                </t-tooltip>
              </div>
            </div>
            <div class="project-configuration-workspace__result-viewer">
              <project-monaco-diff-surface
                v-if="selectedDiffFile && diffViewerReady"
                ref="diffViewerRef"
                class="project-configuration-workspace__monaco-viewer"
                :editor-aria-label="workspaceCopy.diffViewerAriaLabel"
                :language="resolveDiffFileLanguage(selectedDiffFile.kind, selectedDiffFile.path)"
                :modified-key="`diff-modified-${selectedDiffFile.path}`"
                :modified-value="selectedDiffFile.proposed_content"
                :original-key="`diff-original-${selectedDiffFile.path}`"
                :original-value="selectedDiffFile.current_content"
                test-id="configuration-diff-viewer"
              />
              <t-empty v-else :description="workspaceCopy.selectDiffFile" />
            </div>
          </div>
        </div>

        <div v-else-if="validateResult" class="project-configuration-workspace__feedback-panel">
          <t-space size="small" break-line>
            <t-tooltip :content="validateResult.proposed_config_hash" placement="top-left" theme="light">
              <t-tag theme="primary" variant="light-outline">
                {{ t('project.detail.configuration.proposedHash') }}:
                {{ formatWorkspaceHash(validateResult.proposed_config_hash) }}
              </t-tag>
            </t-tooltip>
            <t-tag theme="default" variant="light-outline">
              {{ t('project.detail.configuration.declaredServices') }}:
              {{ validateResult.declared_service_names.join(', ') || '-' }}
            </t-tag>
          </t-space>
          <div v-if="validateResult.warnings?.length" class="project-configuration-workspace__warning-list">
            <t-alert v-for="warning in validateResult.warnings" :key="warning" theme="warning" :message="warning" />
          </div>
          <div class="project-configuration-workspace__result-viewer">
            <project-monaco-surface
              ref="validationViewerRef"
              class="project-configuration-workspace__monaco-viewer"
              :model-value="validateResult.normalized_compose_yaml"
              :editor-aria-label="workspaceCopy.snapshotViewerAriaLabel"
              language="yaml"
              model-key="validation-normalized-yaml"
              :options="readonlyOptions"
              read-only
              test-id="validation-monaco-viewer"
            />
          </div>
        </div>

        <div v-if="resultDialogMode === 'diff'" class="project-configuration-workspace__result-dialog-footer">
          <t-space size="small">
            <t-button
              theme="default"
              variant="outline"
              data-testid="configuration-diff-cancel"
              @click="cancelDiffPreview"
            >
              {{ workspaceCopy.cancelAction }}
            </t-button>
            <t-button
              theme="primary"
              :loading="saveConfirmLoading"
              data-testid="configuration-diff-confirm-save"
              @click="confirmDiffPreview"
            >
              {{ workspaceCopy.confirmSaveAction }}
            </t-button>
          </t-space>
        </div>
      </div>
    </t-dialog>

    <t-dialog
      v-model:visible="dialogState.visible"
      :header="dialogState.title"
      :close-on-overlay-click="false"
      :close-on-esc-keydown="true"
      @close="resolveDialog('cancel')"
    >
      <p class="project-configuration-workspace__dialog-body">{{ dialogState.body }}</p>
      <template #footer>
        <t-space size="small" break-line>
          <t-button
            v-for="button in dialogState.buttons"
            :key="button.result"
            :theme="button.theme"
            :variant="button.variant"
            @click="resolveDialog(button.result)"
          >
            {{ button.label }}
          </t-button>
        </t-space>
      </template>
    </t-dialog>

    <t-dialog
      v-model:visible="annotationDialogState.visible"
      :header="workspaceCopy.annotationAction"
      :close-on-overlay-click="false"
      :confirm-btn="{ content: workspaceCopy.saveAction, loading: annotationDialogState.saving }"
      :cancel-btn="{ content: workspaceCopy.cancelAction }"
      @confirm="saveWorkspaceAnnotation"
      @close="closeWorkspaceAnnotationDialog"
    >
      <t-textarea
        v-model="annotationDialogState.value"
        :autosize="{ minRows: 4, maxRows: 8 }"
        :placeholder="workspaceItemTooltip(annotationDialogState.target)"
      />
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import {
  BrowseIcon,
  BrowseOffIcon,
  CommandIcon,
  Edit1Icon,
  FileCodeIcon,
  FileIcon,
  FolderIcon,
} from 'tdesign-icons-vue-next';
import type { PopupVisibleChangeContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import ContentViewerFrame from '@/shared/components/viewer/ContentViewerFrame.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { createLogger } from '@/utils/logger';

import {
  getProject,
  getProjectConfiguration,
  getProjectConfigurationPreview,
  getProjectFileContent,
  getProjectFiles,
  postProjectConfigurationValidate,
  postProjectDeploy,
  putProjectFileAnnotation,
  putProjectFileContent,
} from '../../api/project';
import ProjectMonacoDiffSurface from '../../components/ProjectMonacoDiffSurface.vue';
import ProjectMonacoSurface from '../../components/ProjectMonacoSurface.vue';
import {
  hasWorkspaceUnsavedChanges,
  normalizeTextBlock,
  normalizeWorkspaceContent,
  type ProjectWorkspaceMonacoLanguage,
  resolveWorkspaceFileName,
  resolveWorkspaceMonacoLanguage,
} from '../../shared/configuration-workspace';
import {
  formatProjectTime,
  projectDriftStatusLabel,
  projectDriftStatusTheme,
  projectRuntimeStatusLabel,
  projectRuntimeStatusTheme,
} from '../../shared/display';
import { useProjectPageContext } from '../../shared/page-context';
import { formatProjectMonacoDebugMessage, isProjectMonacoDebugEnabled } from '../../shared/project-monaco-debug';
import type {
  ProjectConfigurationPreviewResponse,
  ProjectConfigurationValidateResponse,
  ProjectDetailResponseWithLifecycle,
  ProjectWorkspaceFileContentResponse,
  ProjectWorkspaceFileKind,
  ProjectWorkspaceTreeItem,
} from '../../types/project';
import { resolveConfigurationWorkspaceCopy } from './workspace-copy';

defineOptions({
  name: 'ProjectConfigurationWorkspaceIndex',
});

type ResultDialogMode = 'diff' | 'validation';
type DialogResult = 'cancel' | 'continue-disk' | 'discard' | 'save' | 'save-and-continue';
type PendingWorkspaceAction = 'deploy' | 'save' | 'validate';
type WorkspaceDialogButton = {
  label: string;
  result: DialogResult;
  theme: 'default' | 'primary';
  variant: 'base' | 'outline';
};
type WorkspaceListItem = ProjectWorkspaceTreeItem;
type WorkspaceFlatRow = {
  depth: number;
  error: string;
  expanded: boolean;
  item: WorkspaceListItem;
};
type DiffTreeRow = {
  depth: number;
  file: WorkspacePreviewDiffFile | null;
  name: string;
  path: string;
  type: 'directory' | 'file';
};
type WorkspacePreviewDiffFile = {
  changed: boolean;
  current_content: string;
  current_hash: string;
  display_path: string;
  kind: ProjectWorkspaceFileKind;
  path: string;
  proposed_content: string;
  proposed_hash: string;
};
type WorkspacePreviewDiffResult = {
  files: WorkspacePreviewDiffFile[];
  has_changes: boolean;
  warnings: string[];
};
type MonacoViewerHandle = {
  relayout: () => Promise<void>;
};
type WorkspaceOpenFile = {
  content: string;
  readable: boolean;
  editable: boolean;
  error: string;
  fileKind: ProjectWorkspaceFileKind;
  language: ProjectWorkspaceMonacoLanguage;
  loaded: boolean;
  loading: boolean;
  name: string;
  path: string;
  savedContent: string;
  saving: boolean;
  sizeBytes?: number | null;
};

const EDITOR_WIDTH_STORAGE_KEY = 'graft.project.configuration-workspace.sidebar.width';
const EDITOR_HEIGHT_STORAGE_KEY = 'graft.project.configuration-workspace.editor.height.v2';
const SIDEBAR_MAX_WIDTH = 360;
const SIDEBAR_MIN_WIDTH = 208;
const SIDEBAR_DEFAULT_WIDTH = 256;
const SIDEBAR_COLLAPSE_BREAKPOINT = 1024;

const logger = createLogger('project.configuration-workspace');
const route = useRoute();
const { locale, t } = useProjectPageContext();

const workspaceRootRef = ref<HTMLElement | null>(null);
const workspaceShellRef = ref<HTMLElement | null>(null);
const editorStackHostRef = ref<HTMLElement | null>(null);
const workspaceLoading = ref(false);
const workspaceError = ref('');
const workspaceReady = computed(() => Boolean(detailRecord.value && metadata.value && !workspaceError.value));
const browserLoading = ref(false);
const browserError = ref('');
const showHiddenFiles = ref(false);
const rootWorkspaceItems = ref<WorkspaceListItem[]>([]);
const currentWorkspacePath = ref('');
const selectedWorkspacePath = ref('');
const sidebarWidth = ref(resolveStoredSidebarWidth());
const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1440);
const editorViewportHeight = ref(720);
const detailRecord = ref<ProjectDetailResponseWithLifecycle | null>(null);
const metadata = ref<Awaited<ReturnType<typeof getProjectConfiguration>> | null>(null);
const snapshotPreview = ref<ProjectConfigurationPreviewResponse | null>(null);
const snapshotLoading = ref(false);
const snapshotDrawerVisible = ref(false);
const workspaceFullscreen = ref(false);
const resultDialogVisible = ref(false);
const resultDialogMode = ref<ResultDialogMode>('diff');
const resultDialogFullscreen = ref(false);
const diffViewerReady = ref(false);
const diffViewerRef = ref<MonacoViewerHandle | null>(null);
const validationViewerRef = ref<MonacoViewerHandle | null>(null);
const activeFileTabPathForMenu = ref<string | null>(null);
const diffResult = ref<WorkspacePreviewDiffResult | null>(null);
const validateResult = ref<ProjectConfigurationValidateResponse | null>(null);
const diffLoading = ref(false);
const validateLoading = ref(false);
const deployLoading = ref(false);
const saveConfirmLoading = ref(false);
const selectedDiffFilePath = ref('');
const pendingWorkspaceAction = ref<PendingWorkspaceAction | null>(null);
const openTabs = ref<string[]>([]);
const activeTabPath = ref('');
const openFileMap = reactive(new Map<string, WorkspaceOpenFile>());
const directoryChildrenMap = reactive(new Map<string, WorkspaceListItem[]>());
const directoryBrowseStateMap = reactive(new Map<string, { hasMoreHidden: boolean; parentPath: string | null }>());
const directoryErrorMap = reactive(new Map<string, string>());
const directoryLoadingMap = reactive(new Map<string, boolean>());
const expandedDirectoryPaths = ref<string[]>([]);
const dialogState = reactive<{
  body: string;
  buttons: WorkspaceDialogButton[];
  resolver: ((result: DialogResult) => void) | null;
  title: string;
  visible: boolean;
}>({
  body: '',
  buttons: [],
  resolver: null,
  title: '',
  visible: false,
});
const annotationDialogState = reactive<{
  saving: boolean;
  target: WorkspaceListItem | null;
  value: string;
  visible: boolean;
}>({
  saving: false,
  target: null,
  value: '',
  visible: false,
});
let removeSidebarResizeListeners: (() => void) | null = null;

const editorOptions = {
  fontSize: 13,
  lineNumbers: 'on' as const,
  wordWrap: 'off' as const,
};

const readonlyOptions = {
  fontSize: 13,
  lineNumbers: 'on' as const,
  minimap: { enabled: false },
  readOnly: true,
  wordWrap: 'off' as const,
};

const workspaceCopy = computed(() => resolveConfigurationWorkspaceCopy((key) => String(t(key))));
const projectId = computed(() => Number(route.params.id));
const fallbackDisplayName = computed(() => {
  const queryName = typeof route.query.name === 'string' ? route.query.name.trim() : '';
  return queryName;
});
const pageHeaderTitle = computed(() => {
  const suffix = t('project.detail.configuration.title');
  return detailRecord.value?.display_name
    ? `${detailRecord.value.display_name} · ${suffix}`
    : fallbackDisplayName.value
      ? `${fallbackDisplayName.value} · ${suffix}`
      : suffix;
});
const runtimeTheme = computed(() => projectRuntimeStatusTheme(detailRecord.value?.runtime_status));
const runtimeLabel = computed(() => projectRuntimeStatusLabel(t, detailRecord.value?.runtime_status));
const driftTheme = computed(() => projectDriftStatusTheme(metadata.value?.drift_status));
const driftLabel = computed(() =>
  metadata.value?.drift_status ? projectDriftStatusLabel(t, metadata.value.drift_status) : '-',
);
const openTabBuffers = computed(
  () => openTabs.value.map((path) => openFileMap.get(path)).filter(Boolean) as WorkspaceOpenFile[],
);
const activeBuffer = computed(() => (activeTabPath.value ? (openFileMap.get(activeTabPath.value) ?? null) : null));
const dirtyEditableBuffers = computed(() =>
  openTabBuffers.value.filter((tab) => tab.editable && hasWorkspaceUnsavedChanges(tab.content, tab.savedContent)),
);
const canSaveActiveBuffer = computed(() =>
  Boolean(activeBuffer.value?.editable && !activeBuffer.value.saving && dirtyEditableBuffers.value.length),
);
const isSidebarResizable = computed(() => viewportWidth.value > SIDEBAR_COLLAPSE_BREAKPOINT);
const sidebarPaneStyle = computed(() =>
  isSidebarResizable.value ? { width: `${clampSidebarWidth(sidebarWidth.value)}px` } : undefined,
);
const editorFrameHeight = computed(() => Math.max(560, editorViewportHeight.value));
const hasDirtyFiles = computed(() => dirtyEditableBuffers.value.length > 0);
const selectedDiffFile = computed(
  () =>
    diffResult.value?.files.find((file) => file.path === selectedDiffFilePath.value) ??
    diffResult.value?.files[0] ??
    null,
);
const diffFiles = computed(() => diffResult.value?.files ?? []);
const diffTreeRows = computed(() => buildDiffTreeRows(diffFiles.value));
const resultDialogClassName = computed(() =>
  resultDialogFullscreen.value
    ? 'project-configuration-workspace__result-dialog-shell project-configuration-workspace__result-dialog-shell--fullscreen'
    : 'project-configuration-workspace__result-dialog-shell',
);
const resultDialogWidth = computed(() => (resultDialogFullscreen.value ? undefined : 'min(80vw, 1600px)'));
const resultDialogTop = computed(() => (resultDialogFullscreen.value ? 0 : 24));
const resultDialogStyle = computed(() =>
  resultDialogFullscreen.value
    ? {
        height: '100vh',
        maxHeight: '100vh',
        width: '100vw',
      }
    : {
        height: '80vh',
        maxHeight: 'calc(100vh - 48px)',
        width: 'min(80vw, 1600px)',
      },
);
const workspaceItemMap = computed(() => {
  const itemMap = new Map<string, WorkspaceListItem>();
  const appendEntries = (items: WorkspaceListItem[]) => {
    for (const item of items) {
      itemMap.set(item.relative_path, item);
    }
  };

  appendEntries(rootWorkspaceItems.value);
  for (const items of directoryChildrenMap.values()) {
    appendEntries(items);
  }

  return itemMap;
});
const currentWorkspaceDirectoryPath = computed(() => {
  if (selectedWorkspacePath.value) {
    const selectedItem = workspaceItemMap.value.get(selectedWorkspacePath.value);
    if (selectedItem?.node_type === 'directory') {
      return selectedItem.relative_path;
    }
  }

  if (activeBuffer.value?.path) {
    return resolveWorkspaceParentPath(activeBuffer.value.path);
  }

  return currentWorkspacePath.value;
});
const currentWorkspacePathLabel = computed(
  () => currentWorkspaceDirectoryPath.value || workspaceCopy.value.workspaceRootLabel,
);
const workspaceFlatRows = computed(() => flattenWorkspaceRows(rootWorkspaceItems.value, 0));

onMounted(() => {
  window.addEventListener('keydown', handleWorkspaceKeydown);
  window.addEventListener('resize', syncWorkspaceViewport);
  void loadWorkspace();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleWorkspaceKeydown);
  window.removeEventListener('resize', syncWorkspaceViewport);
  stopSidebarResize();
  if (typeof document !== 'undefined') {
    document.body.style.overflow = '';
    document.documentElement.style.overflow = '';
  }
});

watch(showHiddenFiles, () => {
  rootWorkspaceItems.value = [];
  currentWorkspacePath.value = '';
  selectedWorkspacePath.value = '';
  expandedDirectoryPaths.value = [];
  directoryChildrenMap.clear();
  directoryBrowseStateMap.clear();
  directoryErrorMap.clear();
  directoryLoadingMap.clear();
  void loadWorkspaceDirectory('', { root: true });
});

watch(resultDialogVisible, (visible) => {
  if (visible) {
    if (resultDialogMode.value === 'diff') {
      diffViewerReady.value = false;
    }
    queueResultViewerLayout();
  } else {
    diffViewerReady.value = false;
    resultDialogFullscreen.value = false;
    if (resultDialogMode.value === 'diff') {
      pendingWorkspaceAction.value = null;
      saveConfirmLoading.value = false;
    }
  }
});

watch(workspaceFullscreen, (fullscreen) => {
  if (typeof document === 'undefined') {
    return;
  }

  document.body.style.overflow = fullscreen ? 'hidden' : '';
  document.documentElement.style.overflow = fullscreen ? 'hidden' : '';
  void nextTick(() => {
    syncWorkspaceViewport();
  });
});

watch(
  openTabs,
  (nextTabs) => {
    if (!nextTabs.length) {
      activeTabPath.value = '';
      return;
    }
    if (!nextTabs.includes(activeTabPath.value)) {
      activeTabPath.value = nextTabs[nextTabs.length - 1];
    }
  },
  { deep: true },
);

async function loadWorkspace() {
  if (!Number.isFinite(projectId.value)) {
    workspaceError.value = t('project.list.retry');
    return;
  }

  workspaceLoading.value = true;
  workspaceError.value = '';
  try {
    const [detail, configurationMetadata] = await Promise.all([
      getProject(projectId.value),
      getProjectConfiguration(projectId.value),
    ]);
    detailRecord.value = detail;
    metadata.value = configurationMetadata;
    await loadWorkspaceDirectory('', { root: true });
    await nextTick();
    syncWorkspaceViewport();
  } catch (error) {
    logger.error('failed to load project configuration workspace', error);
    workspaceError.value = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
    MessagePlugin.error(workspaceError.value);
  } finally {
    workspaceLoading.value = false;
  }
}

async function loadWorkspaceDirectory(path: string, options?: { root?: boolean }) {
  const normalizedPath = normalizeWorkspacePath(path);
  if (options?.root) {
    browserLoading.value = true;
    browserError.value = '';
  } else {
    directoryLoadingMap.set(normalizedPath, true);
    directoryErrorMap.delete(normalizedPath);
  }

  try {
    const response = await getProjectFiles(projectId.value, {
      path: normalizedPath || undefined,
      show_hidden: showHiddenFiles.value,
    });

    const items = sortWorkspaceItems(response.items ?? []);
    directoryBrowseStateMap.set(normalizedPath, {
      hasMoreHidden: Boolean(response.has_more_hidden),
      parentPath: response.parent_path ?? null,
    });

    if (options?.root) {
      rootWorkspaceItems.value = items;
      currentWorkspacePath.value = response.current_path || '';
    } else {
      directoryChildrenMap.set(normalizedPath, items);
      currentWorkspacePath.value = normalizedPath;
    }

    if (!activeTabPath.value && options?.root) {
      const firstFile = rootWorkspaceItems.value.find((item) => item.node_type === 'file')?.relative_path;
      if (firstFile) {
        await openWorkspaceFile(firstFile);
      }
    }
  } catch (error) {
    const resolvedMessage = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
    if (options?.root) {
      browserError.value = resolvedMessage;
      MessagePlugin.error(browserError.value);
    } else {
      directoryErrorMap.set(normalizedPath, resolvedMessage);
      MessagePlugin.error(resolvedMessage);
    }
  } finally {
    if (options?.root) {
      browserLoading.value = false;
    } else {
      directoryLoadingMap.set(normalizedPath, false);
    }
  }
}

function sortWorkspaceItems(items: ProjectWorkspaceTreeItem[]) {
  return [...items].sort((left, right) => {
    if (left.node_type !== right.node_type) {
      return left.node_type === 'directory' ? -1 : 1;
    }
    return left.name.localeCompare(right.name, undefined, { sensitivity: 'base' });
  });
}

function handleWorkspaceEntry(item: WorkspaceListItem) {
  selectedWorkspacePath.value = item.relative_path;
  if (item.node_type === 'directory') {
    void toggleWorkspaceDirectory(item);
    return;
  }
  if (!item.readable) {
    return;
  }
  void openWorkspaceFile(item.relative_path, item);
}

function workspaceEntryTestId(item: WorkspaceListItem) {
  const raw = item.relative_path || item.name || 'root';
  return `workspace-entry-${
    raw
      .replace(/[^a-z0-9]+/gi, '-')
      .replace(/^-+|-+$/g, '')
      .toLowerCase() || 'root'
  }`;
}

function workspaceAnnotationTestId(item: WorkspaceListItem) {
  return `${workspaceEntryTestId(item)}-annotation`;
}

function workspaceFileTabMenuTestId(path: string) {
  return `workspace-file-tab-menu-${workspaceEntryTestId({ relative_path: path, name: path } as WorkspaceListItem)}`;
}

function workspaceFileTabMenuItemTestId(path: string, action: string) {
  return `${workspaceFileTabMenuTestId(path)}-${action}`;
}

function diffFileTestId(path: string) {
  return `configuration-diff-file-${workspaceEntryTestId({ relative_path: path, name: path } as WorkspaceListItem)}`;
}

async function toggleWorkspaceDirectory(item: WorkspaceListItem) {
  if (item.node_type !== 'directory') {
    return;
  }

  const path = item.relative_path;
  currentWorkspacePath.value = path;
  selectedWorkspacePath.value = path;
  if (expandedDirectoryPaths.value.includes(path)) {
    expandedDirectoryPaths.value = expandedDirectoryPaths.value.filter((value) => value !== path);
    return;
  }

  expandedDirectoryPaths.value = [...expandedDirectoryPaths.value, path];
  if (!directoryChildrenMap.has(path) && item.has_children) {
    await loadWorkspaceDirectory(path);
  }
}

function flattenWorkspaceRows(items: WorkspaceListItem[], depth: number) {
  const rows: WorkspaceFlatRow[] = [];

  for (const item of items) {
    const path = item.relative_path;
    const expanded = item.node_type === 'directory' && expandedDirectoryPaths.value.includes(path);
    rows.push({
      depth,
      error: directoryErrorMap.get(path) ?? '',
      expanded,
      item,
    });

    if (expanded) {
      rows.push(...flattenWorkspaceRows(directoryChildrenMap.get(path) ?? [], depth + 1));
    }
  }

  return rows;
}

function workspaceItemTooltip(item?: WorkspaceListItem | null) {
  if (!item) {
    return '';
  }
  const projectNote = String(item.project_note ?? '').trim();
  if (projectNote) {
    return projectNote;
  }

  const tooltip = String(item.tooltip ?? '').trim();
  return tooltip || '';
}

function isWorkspaceItemActive(item: WorkspaceListItem) {
  return (
    (activeTabPath.value && activeTabPath.value === item.relative_path) ||
    selectedWorkspacePath.value === item.relative_path
  );
}

function handleWorkspaceAnnotation(item: WorkspaceListItem) {
  annotationDialogState.target = item;
  annotationDialogState.value = String(item.project_note ?? '').trim();
  annotationDialogState.visible = true;
}

function closeWorkspaceAnnotationDialog() {
  annotationDialogState.visible = false;
  annotationDialogState.saving = false;
  annotationDialogState.target = null;
  annotationDialogState.value = '';
}

async function saveWorkspaceAnnotation() {
  const target = annotationDialogState.target;
  if (!target) {
    closeWorkspaceAnnotationDialog();
    return;
  }

  annotationDialogState.saving = true;
  try {
    const annotation = annotationDialogState.value.trim();
    const updatedItem = await putProjectFileAnnotation(
      projectId.value,
      { path: target.relative_path },
      { annotation: annotation || null },
    );
    patchWorkspaceItem(updatedItem);
    closeWorkspaceAnnotationDialog();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, workspaceCopy.value.annotationSaveFailed));
  } finally {
    annotationDialogState.saving = false;
  }
}

function patchWorkspaceItem(nextItem: WorkspaceListItem) {
  if (!nextItem?.relative_path) {
    return;
  }

  rootWorkspaceItems.value = replaceWorkspaceItem(rootWorkspaceItems.value, nextItem);
  for (const [path, items] of directoryChildrenMap.entries()) {
    directoryChildrenMap.set(path, replaceWorkspaceItem(items, nextItem));
  }
}

function replaceWorkspaceItem(items: WorkspaceListItem[], nextItem: WorkspaceListItem) {
  return items.map((item) => (item.relative_path === nextItem.relative_path ? nextItem : item));
}

async function openWorkspaceFile(path: string, source?: WorkspaceListItem) {
  if (!path) {
    return;
  }

  if (!openFileMap.has(path)) {
    openFileMap.set(path, {
      content: '',
      readable: Boolean(source?.readable ?? true),
      editable: Boolean(source?.editable),
      error: '',
      fileKind: source?.file_kind || 'text',
      language: resolveWorkspaceMonacoLanguage({
        fileKind: source?.file_kind,
        languageHint: source?.language_hint,
        path,
      }),
      loaded: false,
      loading: false,
      name: resolveWorkspaceFileName(path),
      path,
      savedContent: '',
      saving: false,
      sizeBytes: source?.size_bytes ?? null,
    });
    openTabs.value = [...openTabs.value, path];
  }

  const current = openFileMap.get(path);
  const shouldDelayActivation = Boolean(current && !current.loaded && !current.loading);

  if (!shouldDelayActivation) {
    activeTabPath.value = path;
  }

  const resolvedPath = await ensureWorkspaceFileLoaded(path, source);
  activeTabPath.value = resolvedPath ?? path;
}

async function ensureWorkspaceFileLoaded(path: string, source?: WorkspaceListItem) {
  const current = openFileMap.get(path);
  if (!current || current.loading || current.loaded) {
    return current?.path ?? path;
  }

  current.loading = true;
  current.error = '';
  try {
    const response = await getProjectFileContent(projectId.value, { path });
    return hydrateOpenFileFromResponse(path, response, source);
  } catch (error) {
    current.error = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    current.loading = false;
  }

  return current.path ?? path;
}

function hydrateOpenFileFromResponse(
  requestedPath: string,
  response: ProjectWorkspaceFileContentResponse,
  source?: WorkspaceListItem,
) {
  const path = response.relative_path || requestedPath;
  const current =
    openFileMap.get(requestedPath) ??
    ({
      content: '',
      readable: response.readable,
      editable: response.editable,
      error: '',
      fileKind: response.file_kind,
      language: resolveWorkspaceMonacoLanguage({
        fileKind: response.file_kind,
        languageHint: response.language_hint,
        path,
      }),
      loaded: false,
      loading: false,
      name: resolveWorkspaceFileName(path),
      path,
      savedContent: '',
      saving: false,
    } satisfies WorkspaceOpenFile);

  current.content = normalizeWorkspaceContent(response.content);
  current.readable = response.readable;
  current.editable = response.editable;
  current.error = '';
  current.fileKind = response.file_kind || source?.file_kind || 'text';
  current.language = resolveWorkspaceMonacoLanguage({
    fileKind: current.fileKind,
    languageHint: response.language_hint ?? source?.language_hint,
    path,
  });
  current.loaded = true;
  current.loading = false;
  current.name = resolveWorkspaceFileName(path);
  current.path = path;
  current.savedContent = normalizeWorkspaceContent(response.content);
  current.sizeBytes = response.size_bytes ?? source?.size_bytes ?? null;
  openFileMap.set(path, current);

  if (path !== requestedPath) {
    openFileMap.delete(requestedPath);
    openTabs.value = openTabs.value.map((item) => (item === requestedPath ? path : item));
    if (activeTabPath.value === requestedPath) {
      activeTabPath.value = path;
    }
  }

  return path;
}

function isFileDirty(path: string) {
  const current = openFileMap.get(path);
  return current ? hasWorkspaceUnsavedChanges(current.content, current.savedContent) : false;
}

async function saveActiveFile() {
  if (!canSaveActiveBuffer.value) {
    return;
  }
  await previewBeforeSave('save');
}

async function saveWorkspaceFile(path: string, options?: { silent?: boolean }) {
  const current = openFileMap.get(path);
  if (!current || !current.editable || current.saving) {
    return false;
  }

  current.saving = true;
  try {
    const normalizedContent = normalizeTextBlock(current.content);
    await putProjectFileContent(projectId.value, { path }, { content: normalizedContent });
    current.content = normalizedContent;
    current.savedContent = normalizedContent;
    if (!options?.silent) {
      MessagePlugin.success(workspaceCopy.value.saveSuccess);
    }
    return true;
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, workspaceCopy.value.saveFailed));
    return false;
  } finally {
    current.saving = false;
  }
}

async function saveDirtyFiles() {
  const dirtyPaths = dirtyEditableBuffers.value.map((tab) => tab.path);
  if (!dirtyPaths.length) {
    return true;
  }

  for (const path of dirtyPaths) {
    const saved = await saveWorkspaceFile(path, { silent: true });
    if (!saved) {
      return false;
    }
  }
  MessagePlugin.success(workspaceCopy.value.saveSuccess);
  return true;
}

async function reloadActiveFile() {
  if (!activeBuffer.value) {
    return;
  }

  await reloadWorkspaceFile(activeBuffer.value.path);
}

async function reloadWorkspaceFile(path: string) {
  const buffer = openFileMap.get(path);
  if (!buffer) {
    return;
  }

  if (isFileDirty(path)) {
    const action = await openDialog({
      body: workspaceCopy.value.reloadConfirmBody,
      buttons: [
        { label: workspaceCopy.value.discardAction, result: 'discard', theme: 'primary', variant: 'base' },
        { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
      ],
      title: workspaceCopy.value.reloadConfirmTitle,
    });
    if (action !== 'discard') {
      return;
    }
  }

  buffer.loading = true;
  buffer.error = '';
  try {
    const response = await getProjectFileContent(projectId.value, { path: buffer.path });
    hydrateOpenFileFromResponse(buffer.path, response);
  } catch (error) {
    buffer.error = resolveLocalizedErrorMessage(t, error, t('project.list.retry'));
  } finally {
    buffer.loading = false;
  }
}

async function handleCloseTab(path: string) {
  const closed = await closeWorkspaceTabs([path], { skipBatchDirtyPrompt: true });
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function closeWorkspaceTabs(paths: string[], options?: { skipBatchDirtyPrompt?: boolean }) {
  const uniquePaths = [...new Set(paths)].filter((path) => openFileMap.has(path));
  if (!uniquePaths.length) {
    return false;
  }

  if (!options?.skipBatchDirtyPrompt) {
    const dirtyPaths = uniquePaths.filter((path) => isFileDirty(path));
    if (dirtyPaths.length) {
      const action = await openDialog({
        body: workspaceCopy.value.dirtyProjectActionBody,
        buttons: [
          { label: workspaceCopy.value.saveThenContinueAction, result: 'save', theme: 'primary', variant: 'base' },
          { label: workspaceCopy.value.discardAction, result: 'discard', theme: 'default', variant: 'outline' },
          { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
        ],
        title: workspaceCopy.value.dirtyProjectActionTitle,
      });
      if (action === 'cancel') {
        return false;
      }
      if (action === 'save') {
        const saved = await saveTabsByPaths(dirtyPaths);
        if (!saved) {
          return false;
        }
      }
    }
  }

  if (uniquePaths.length === 1) {
    return closeSingleWorkspaceTab(uniquePaths[0]);
  }

  const closedPathSet = new Set(uniquePaths);
  const currentTabs = [...openTabs.value];
  const currentActivePath = activeTabPath.value;
  const activeIndex = currentTabs.findIndex((item) => item === currentActivePath);
  const nextTabs = currentTabs.filter((path) => !closedPathSet.has(path));

  uniquePaths.forEach((path) => {
    openFileMap.delete(path);
  });

  openTabs.value = nextTabs;
  if (!currentActivePath || !closedPathSet.has(currentActivePath)) {
    return true;
  }

  activeTabPath.value = resolveNextWorkspaceTabAfterClose(currentTabs, closedPathSet, activeIndex);
  return true;
}

function resolveNextWorkspaceTabAfterClose(currentTabs: string[], closedPathSet: Set<string>, activeIndex: number) {
  const nextRight = currentTabs.slice(activeIndex + 1).find((path) => !closedPathSet.has(path));
  if (nextRight) {
    return nextRight;
  }

  const nextLeft = [...currentTabs.slice(0, Math.max(activeIndex, 0))]
    .reverse()
    .find((path) => !closedPathSet.has(path));
  return nextLeft || '';
}

async function closeSingleWorkspaceTab(path: string) {
  const current = openFileMap.get(path);
  if (!current) {
    return false;
  }

  if (isFileDirty(path)) {
    const action = await openDialog({
      body: workspaceCopy.value.dirtyCloseBody,
      buttons: [
        { label: workspaceCopy.value.saveAction, result: 'save', theme: 'primary', variant: 'base' },
        { label: workspaceCopy.value.discardAction, result: 'discard', theme: 'default', variant: 'outline' },
        { label: workspaceCopy.value.cancelAction, result: 'cancel', theme: 'default', variant: 'outline' },
      ],
      title: workspaceCopy.value.dirtyCloseTitle,
    });
    if (action === 'cancel') {
      return false;
    }
    if (action === 'save') {
      const saved = await saveWorkspaceFile(path);
      if (!saved) {
        return false;
      }
    }
  }

  openFileMap.delete(path);
  openTabs.value = openTabs.value.filter((item) => item !== path);
  return true;
}

async function saveTabsByPaths(paths: string[]) {
  const uniquePaths = [...new Set(paths)].filter((path) => isFileDirty(path));
  if (!uniquePaths.length) {
    return true;
  }

  for (const path of uniquePaths) {
    const saved = await saveWorkspaceFile(path, { silent: true });
    if (!saved) {
      return false;
    }
  }

  MessagePlugin.success(workspaceCopy.value.saveSuccess);
  return true;
}

function hasClosableFileTabsAhead(path: string) {
  const index = openTabs.value.indexOf(path);
  return index > 0;
}

function hasClosableFileTabsBehind(path: string) {
  const index = openTabs.value.indexOf(path);
  return index !== -1 && index < openTabs.value.length - 1;
}

function hasClosableOtherFileTabs(path: string) {
  return openTabs.value.length > 1 && openTabs.value.some((item) => item !== path);
}

const hasClosableFileTabs = computed(() => openTabs.value.length > 0);

async function handleRefreshFileTab(path: string) {
  await reloadWorkspaceFile(path);
  activeFileTabPathForMenu.value = null;
}

async function handleCloseFileTabsAhead(path: string) {
  const targetIndex = openTabs.value.indexOf(path);
  if (targetIndex <= 0) {
    activeFileTabPathForMenu.value = null;
    return;
  }

  const closed = await closeWorkspaceTabs(openTabs.value.slice(0, targetIndex));
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function handleCloseFileTabsBehind(path: string) {
  const targetIndex = openTabs.value.indexOf(path);
  if (targetIndex === -1 || targetIndex >= openTabs.value.length - 1) {
    activeFileTabPathForMenu.value = null;
    return;
  }

  const closed = await closeWorkspaceTabs(openTabs.value.slice(targetIndex + 1));
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function handleCloseOtherFileTabs(path: string) {
  const closed = await closeWorkspaceTabs(openTabs.value.filter((item) => item !== path));
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

async function handleCloseAllFileTabs() {
  const closed = await closeWorkspaceTabs([...openTabs.value]);
  if (closed) {
    activeFileTabPathForMenu.value = null;
  }
}

function handleFileTabMenuClick(visible: boolean, ctx: PopupVisibleChangeContext, path: string) {
  if (visible) {
    activeFileTabPathForMenu.value = path;
    return;
  }

  if (activeFileTabPathForMenu.value === path || ctx.trigger === 'document') {
    activeFileTabPathForMenu.value = null;
  }
}

async function runProjectValidate() {
  if (hasDirtyFiles.value) {
    await previewBeforeSave('validate');
    return;
  }

  await executeProjectValidate();
}

async function executeProjectValidate() {
  validateLoading.value = true;
  try {
    validateResult.value = await postProjectConfigurationValidate(projectId.value);
    resultDialogMode.value = 'validation';
    resultDialogVisible.value = true;
    MessagePlugin.success(t('project.detail.configuration.validateSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.validateFailed')));
  } finally {
    validateLoading.value = false;
  }
}

function toggleResultDialogFullscreen() {
  resultDialogFullscreen.value = !resultDialogFullscreen.value;
  queueResultViewerLayout();
}

function handleResultDialogOpened() {
  if (resultDialogMode.value === 'diff') {
    logWorkspaceDiffDebug('dialog-opened', {
      diffViewerReady: diffViewerReady.value,
      fileCount: diffFiles.value.length,
      selectedPath: selectedDiffFile.value?.path ?? '',
    });
    diffViewerReady.value = true;
  }
  queueResultViewerLayout();
}

function queueResultViewerLayout() {
  void nextTick(async () => {
    const layoutTasks: Array<Promise<void>> = [];
    if (diffViewerRef.value && typeof diffViewerRef.value.relayout === 'function') {
      layoutTasks.push(diffViewerRef.value.relayout());
    }
    if (validationViewerRef.value && typeof validationViewerRef.value.relayout === 'function') {
      layoutTasks.push(validationViewerRef.value.relayout());
    }
    await Promise.allSettled(layoutTasks);
  });
}

function logWorkspaceDiffDebug(event: string, detail: Record<string, unknown>) {
  if (!isProjectMonacoDebugEnabled()) {
    return;
  }

  logger.warn(`[ConfigurationWorkspaceDiff] ${formatProjectMonacoDebugMessage(event, detail)}`, detail);
}

async function runProjectDeploy() {
  if (hasDirtyFiles.value) {
    await previewBeforeSave('deploy');
    return;
  }

  await executeProjectDeploy();
}

async function executeProjectDeploy() {
  deployLoading.value = true;
  try {
    const response = await postProjectDeploy(projectId.value);
    MessagePlugin.success(response.message || t('project.detail.configuration.deploySuccess'));
    diffResult.value = null;
    resultDialogVisible.value = false;
    validateResult.value = null;
    snapshotPreview.value = null;
    await loadWorkspaceDirectory('', { root: true });
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.detail.configuration.deployFailed')));
  } finally {
    deployLoading.value = false;
  }
}

function buildDirtyDiffFiles(paths?: string[]): WorkspacePreviewDiffFile[] {
  const targetPathSet = paths?.length ? new Set(paths) : null;
  return dirtyEditableBuffers.value
    .filter((tab) => !targetPathSet || targetPathSet.has(tab.path))
    .map((tab) => {
      const currentContent = normalizeTextBlock(tab.savedContent);
      const proposedContent = normalizeTextBlock(tab.content);
      const path = normalizeWorkspacePath(tab.path);
      return {
        changed: currentContent !== proposedContent,
        current_content: currentContent,
        current_hash: hashWorkspaceContent(currentContent),
        display_path: path,
        kind: tab.fileKind,
        path,
        proposed_content: proposedContent,
        proposed_hash: hashWorkspaceContent(proposedContent),
      } satisfies WorkspacePreviewDiffFile;
    })
    .filter((file) => file.changed);
}

async function previewBeforeSave(action: PendingWorkspaceAction) {
  const files = buildDirtyDiffFiles();
  logWorkspaceDiffDebug('preview-before-save', {
    action,
    diffFileCount: files.length,
    dirtyBufferCount: dirtyEditableBuffers.value.length,
  });
  if (!files.length) {
    MessagePlugin.info(workspaceCopy.value.diffEmptyDirectSaveHint);
    const saved = await saveDirtyFiles();
    if (!saved) {
      return false;
    }
    if (action === 'validate') {
      await executeProjectValidate();
    } else if (action === 'deploy') {
      await executeProjectDeploy();
    }
    return true;
  }

  diffResult.value = {
    files,
    has_changes: files.length > 0,
    warnings: [],
  };
  selectedDiffFilePath.value = files[0]?.path || '';
  logWorkspaceDiffDebug('preview-diff-selected', {
    currentHash: files[0]?.current_hash ?? '',
    currentLength: files[0]?.current_content.length ?? 0,
    diffViewerReady: diffViewerReady.value,
    path: files[0]?.path ?? '',
    proposedHash: files[0]?.proposed_hash ?? '',
    proposedLength: files[0]?.proposed_content.length ?? 0,
  });
  validateResult.value = null;
  pendingWorkspaceAction.value = action;
  resultDialogMode.value = 'diff';
  resultDialogVisible.value = true;
  return false;
}

function cancelDiffPreview() {
  resultDialogVisible.value = false;
}

async function confirmDiffPreview() {
  if (!pendingWorkspaceAction.value || saveConfirmLoading.value) {
    return;
  }

  saveConfirmLoading.value = true;
  const action = pendingWorkspaceAction.value;
  try {
    const saved = await saveDirtyFiles();
    if (!saved) {
      return;
    }

    resultDialogVisible.value = false;
    diffResult.value = null;
    selectedDiffFilePath.value = '';
    pendingWorkspaceAction.value = null;

    if (action === 'validate') {
      await executeProjectValidate();
    } else if (action === 'deploy') {
      await executeProjectDeploy();
    }
  } finally {
    saveConfirmLoading.value = false;
  }
}

async function openSnapshotDrawer() {
  snapshotDrawerVisible.value = true;
  if (snapshotPreview.value || !Number.isFinite(projectId.value)) {
    return;
  }
  snapshotLoading.value = true;
  try {
    snapshotPreview.value = await getProjectConfigurationPreview(projectId.value);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.list.retry')));
  } finally {
    snapshotLoading.value = false;
  }
}

function resolveDiffFileLanguage(kind: string | undefined, path: string) {
  return resolveWorkspaceMonacoLanguage({
    fileKind: kind === 'compose' || kind === 'env' ? kind : 'config',
    path,
  });
}

function formatWorkspaceHash(value?: string | null) {
  const normalized = String(value ?? '').trim();
  if (!normalized) {
    return '-';
  }
  if (normalized.length <= 14) {
    return normalized;
  }
  return `${normalized.slice(0, 6)}...${normalized.slice(-6)}`;
}

function diffFileName(path: string) {
  const normalized = String(path).trim().replaceAll('\\', '/');
  const segments = normalized.split('/').filter(Boolean);
  return segments[segments.length - 1] || normalized || '-';
}

function hashWorkspaceContent(value: string) {
  let hash = 2166136261;
  for (const char of value) {
    hash ^= char.charCodeAt(0);
    hash = Math.imul(hash, 16777619);
  }
  return `ws-${(hash >>> 0).toString(16).padStart(8, '0')}`;
}

function buildDiffTreeRows(files: WorkspacePreviewDiffFile[]) {
  const directoryPaths = new Set<string>();
  const fileMap = new Map<string, WorkspacePreviewDiffFile>();

  for (const file of files) {
    const normalizedPath = normalizeWorkspacePath(file.path);
    fileMap.set(normalizedPath, file);

    const segments = normalizedPath.split('/').filter(Boolean);
    let currentPath = '';
    for (const segment of segments.slice(0, -1)) {
      currentPath = currentPath ? `${currentPath}/${segment}` : segment;
      directoryPaths.add(currentPath);
    }
  }

  const appendRows = (basePath: string, depth: number): DiffTreeRow[] => {
    const childDirectories = [...directoryPaths]
      .filter((path) => resolveWorkspaceParentPath(path) === basePath)
      .sort((left, right) => left.localeCompare(right, undefined, { sensitivity: 'base' }));
    const childFiles = [...fileMap.entries()]
      .filter(([path]) => resolveWorkspaceParentPath(path) === basePath)
      .sort(([left], [right]) => left.localeCompare(right, undefined, { sensitivity: 'base' }));

    const rows: DiffTreeRow[] = [];
    for (const directoryPath of childDirectories) {
      rows.push({
        depth,
        file: null,
        name: diffFileName(directoryPath),
        path: directoryPath,
        type: 'directory',
      });
      rows.push(...appendRows(directoryPath, depth + 1));
    }

    for (const [path, file] of childFiles) {
      rows.push({
        depth,
        file,
        name: diffFileName(file.display_path || path),
        path,
        type: 'file',
      });
    }

    return rows;
  };

  return appendRows('', 0);
}

function openDialog(config: { body: string; buttons: WorkspaceDialogButton[]; title: string }) {
  if (dialogState.resolver) {
    dialogState.resolver('cancel');
  }

  dialogState.body = config.body;
  dialogState.buttons = config.buttons;
  dialogState.title = config.title;
  dialogState.visible = true;

  return new Promise<DialogResult>((resolve) => {
    dialogState.resolver = resolve;
  });
}

function resolveDialog(result: DialogResult) {
  if (!dialogState.resolver) {
    return;
  }

  const resolver = dialogState.resolver;
  dialogState.body = '';
  dialogState.buttons = [];
  dialogState.title = '';
  dialogState.visible = false;
  dialogState.resolver = null;
  resolver(result);
}

function handleWorkspaceKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && workspaceFullscreen.value) {
    workspaceFullscreen.value = false;
    return;
  }

  const root = workspaceRootRef.value;
  if (!root || !activeBuffer.value || !canSaveActiveBuffer.value) {
    return;
  }

  const target = event.target;
  if (!(target instanceof Node) || !root.contains(target)) {
    return;
  }

  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
    event.preventDefault();
    void saveActiveFile();
  }
}

function normalizeWorkspacePath(path: string) {
  return String(path || '')
    .replace(/\\/g, '/')
    .replace(/^\/+|\/+$/g, '');
}

function resolveWorkspaceParentPath(path: string) {
  const normalizedPath = normalizeWorkspacePath(path);
  if (!normalizedPath.includes('/')) {
    return '';
  }

  return normalizedPath.split('/').slice(0, -1).join('/');
}

function resolveStoredSidebarWidth() {
  if (typeof window === 'undefined') {
    return SIDEBAR_DEFAULT_WIDTH;
  }

  try {
    const raw = window.localStorage.getItem(EDITOR_WIDTH_STORAGE_KEY);
    const parsed = Number(raw);
    return Number.isFinite(parsed) ? clampSidebarWidth(parsed) : SIDEBAR_DEFAULT_WIDTH;
  } catch {
    return SIDEBAR_DEFAULT_WIDTH;
  }
}

function clampSidebarWidth(value: number) {
  const shellWidth = workspaceShellRef.value?.clientWidth || 1280;
  const maxWidth = Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, shellWidth - 420));
  return Math.max(SIDEBAR_MIN_WIDTH, Math.min(maxWidth, Math.round(value)));
}

function writeStoredSidebarWidth(value: number) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(EDITOR_WIDTH_STORAGE_KEY, String(clampSidebarWidth(value)));
  } catch {
    return;
  }
}

function syncWorkspaceViewport() {
  if (typeof window === 'undefined') {
    return;
  }

  viewportWidth.value = window.innerWidth;
  sidebarWidth.value = clampSidebarWidth(sidebarWidth.value);
  syncEditorViewportHeight();
}

function syncEditorViewportHeight() {
  if (typeof window === 'undefined') {
    return;
  }

  const editorTop = editorStackHostRef.value?.getBoundingClientRect().top ?? 320;
  editorViewportHeight.value = Math.max(560, Math.floor(window.innerHeight - editorTop - 40));
}

function startSidebarResize(event: PointerEvent) {
  if (!isSidebarResizable.value || typeof window === 'undefined') {
    return;
  }

  stopSidebarResize();

  const shellBounds = workspaceShellRef.value?.getBoundingClientRect();
  if (!shellBounds) {
    return;
  }

  const handlePointerMove = (moveEvent: PointerEvent) => {
    sidebarWidth.value = clampSidebarWidth(moveEvent.clientX - shellBounds.left);
  };

  const handlePointerUp = () => {
    writeStoredSidebarWidth(sidebarWidth.value);
    stopSidebarResize();
  };

  document.body.classList.add('graft-resizing');
  window.addEventListener('pointermove', handlePointerMove);
  window.addEventListener('pointerup', handlePointerUp, { once: true });
  removeSidebarResizeListeners = () => {
    document.body.classList.remove('graft-resizing');
    window.removeEventListener('pointermove', handlePointerMove);
    window.removeEventListener('pointerup', handlePointerUp);
  };
  handlePointerMove(event);
}

function stopSidebarResize() {
  removeSidebarResizeListeners?.();
  removeSidebarResizeListeners = null;
}
</script>
<style scoped lang="less">
.project-configuration-workspace {
  min-width: 0;

  --graft-workspace-editor-surface: color-mix(
    in srgb,
    var(--td-bg-color-container) 84%,
    var(--graft-shell-content-bg, var(--td-bg-color-page)) 16%
  );
  --graft-workspace-editor-surface-raised: color-mix(
    in srgb,
    var(--graft-workspace-editor-surface) 82%,
    var(--td-bg-color-container-hover) 18%
  );
  --graft-workspace-editor-surface-muted: color-mix(
    in srgb,
    var(--graft-workspace-editor-surface) 78%,
    var(--graft-shell-content-bg, var(--td-bg-color-page)) 22%
  );
  --graft-workspace-editor-border: color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  --graft-workspace-editor-foreground: var(--td-text-color-primary);
  --graft-workspace-editor-foreground-muted: color-mix(
    in srgb,
    var(--td-text-color-secondary) 92%,
    var(--td-text-color-primary)
  );
  --graft-workspace-editor-foreground-subtle: color-mix(in srgb, var(--td-text-color-placeholder) 78%, transparent);
  --graft-workspace-editor-accent: var(--td-brand-color-6);
  --graft-workspace-editor-line-highlight: color-mix(in srgb, var(--td-brand-color-6) 13%, transparent);
  --graft-workspace-editor-selection: color-mix(in srgb, var(--td-brand-color-6) 28%, transparent);
  --graft-workspace-editor-selection-inactive: color-mix(in srgb, var(--td-brand-color-6) 18%, transparent);
  --graft-workspace-editor-indent-guide: color-mix(in srgb, var(--td-text-color-placeholder) 24%, transparent);
  --graft-workspace-editor-indent-guide-active: color-mix(
    in srgb,
    var(--td-brand-color-6) 30%,
    var(--td-component-stroke)
  );
  --graft-workspace-editor-find-match: color-mix(in srgb, var(--td-brand-color-6) 24%, var(--td-warning-color-1));
  --graft-workspace-editor-find-match-border: color-mix(
    in srgb,
    var(--td-brand-color-6) 52%,
    var(--td-component-stroke)
  );
  --graft-workspace-editor-diff-added: color-mix(in srgb, var(--td-success-color-5) 18%, transparent);
  --graft-workspace-editor-diff-removed: color-mix(in srgb, var(--td-error-color-5) 18%, transparent);
  --graft-workspace-tab-indicator: var(--td-brand-color-6);
  --graft-workspace-tab-hover: color-mix(in srgb, var(--td-brand-color-6) 8%, transparent);
}

.project-configuration-workspace--fullscreen {
  min-width: 0;
}

.project-configuration-workspace__summary-strip,
.project-configuration-workspace__main-grid {
  margin-top: var(--graft-density-gap-16);
}

.project-configuration-workspace__section-head {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-configuration-workspace__section-head h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.project-configuration-workspace__summary-strip .project-configuration-workspace__section-head p,
.project-configuration-workspace__result-dialog-header .project-configuration-workspace__section-head p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}

.project-configuration-workspace__main-grid {
  display: grid;
  gap: var(--graft-density-gap-4);
  grid-template-columns: minmax(0, auto) auto minmax(0, 1fr);
  min-height: 0;
}

.project-configuration-workspace--fullscreen .project-configuration-workspace__summary-strip {
  display: none;
}

.project-configuration-workspace--fullscreen .project-configuration-workspace__main-grid {
  background: var(--graft-shell-content-bg, var(--td-bg-color-page));
  height: calc(100vh - 32px);
  inset: var(--graft-density-gap-16);
  margin-top: 0;
  min-height: calc(100vh - 32px);
  padding: 0;
  position: fixed;
  z-index: 4400;
}

.project-configuration-workspace--fullscreen .project-configuration-workspace__sidebar,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack,
.project-configuration-workspace--fullscreen .project-configuration-workspace__browser-card,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-surface,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stage,
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-loading {
  height: 100%;
}

.project-configuration-workspace--fullscreen
  .project-configuration-workspace__editor-stack
  :deep(.content-viewer-frame),
.project-configuration-workspace--fullscreen
  .project-configuration-workspace__editor-stack
  :deep(.content-viewer-frame__panel),
.project-configuration-workspace--fullscreen
  .project-configuration-workspace__editor-stack
  :deep(.content-viewer-frame__surface),
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack :deep(.t-loading__parent),
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack :deep(.t-loading__content),
.project-configuration-workspace--fullscreen .project-configuration-workspace__editor-stack :deep(.t-loading__wrap) {
  height: 100%;
}

.project-configuration-workspace__sidebar,
.project-configuration-workspace__browser-card,
.project-configuration-workspace__diff-sidebar,
.project-configuration-workspace__diff-stage,
.project-configuration-workspace__diff-surface,
.project-configuration-workspace__editor-stack,
.project-configuration-workspace__feedback-panel,
.project-configuration-workspace__result-dialog,
.project-configuration-workspace__result-viewer,
.project-configuration-workspace__readonly-viewer,
.project-configuration-workspace__drawer-viewer {
  min-height: 0;
  min-width: 0;
}

.project-configuration-workspace__browser-alert,
.project-configuration-workspace__editor-alert {
  margin-bottom: var(--graft-density-gap-12);
}

.project-configuration-workspace__browser-card {
  background:
    radial-gradient(circle at top left, color-mix(in srgb, var(--td-brand-color-6) 7%, transparent), transparent 38%),
    var(--graft-workspace-editor-surface-muted);
  border-color: var(--graft-workspace-editor-border);
  box-shadow: none;
  height: 100%;
}

.project-configuration-workspace__browser-card :deep(.t-card__header) {
  padding-bottom: var(--graft-density-gap-8);
}

.project-configuration-workspace__browser-card :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding-top: var(--graft-density-gap-8);
}

.project-configuration-workspace__tree-toolbar-button {
  color: var(--td-text-color-secondary);
}

.project-configuration-workspace__tree-toolbar-button:hover {
  color: var(--td-brand-color-6);
}

.project-configuration-workspace__tree {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-height: 0;
  overflow: auto;
}

.project-configuration-workspace__tree-row {
  align-items: center;
  border-radius: var(--td-radius-default);
  display: grid;
  gap: var(--graft-density-gap-4);
  grid-template-columns: 18px minmax(0, 1fr) auto;
  min-width: 0;
  padding-left: calc(var(--workspace-tree-depth, 0) * var(--graft-density-gap-14));
  transition: background-color 0.2s ease;
}

.project-configuration-workspace__tree-row--active {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
}

.project-configuration-workspace__tree-row--readonly {
  opacity: 0.68;
}

.project-configuration-workspace__tree-expander,
.project-configuration-workspace__tree-expander-placeholder {
  align-items: center;
  color: var(--td-text-color-placeholder);
  display: inline-flex;
  height: 24px;
  justify-content: center;
  width: 18px;
}

.project-configuration-workspace__tree-expander {
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  cursor: pointer;
  padding: 0;
}

.project-configuration-workspace__tree-expander:hover {
  background: color-mix(in srgb, var(--td-brand-color-6) 10%, transparent);
  color: var(--td-text-color-primary);
}

.project-configuration-workspace__tree-expander-icon {
  font-size: var(--td-font-size-body-small);
  line-height: 1;
}

.project-configuration-workspace__tree-entry {
  align-items: center;
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-default);
  color: inherit;
  cursor: pointer;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-height: 30px;
  min-width: 0;
  padding: 0 var(--graft-density-gap-8) 0 0;
  text-align: left;
  width: 100%;
}

.project-configuration-workspace__tree-entry:hover {
  color: var(--td-text-color-primary);
}

.project-configuration-workspace__browser-icon {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  flex: 0 0 auto;
  height: 16px;
  justify-content: center;
  line-height: 0;
  width: 16px;
}

.project-configuration-workspace__browser-icon :deep(svg) {
  height: 16px;
  width: 16px;
}

.project-configuration-workspace__docker-icon {
  color: #2496ed;
  display: inline-flex;
  height: 16px;
  width: 16px;
}

.project-configuration-workspace__docker-icon svg {
  fill: currentcolor;
  height: 100%;
  width: 100%;
}

.project-configuration-workspace__browser-main {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.project-configuration-workspace__browser-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-configuration-workspace__browser-meta {
  color: var(--td-text-color-placeholder);
  display: flex;
  flex-wrap: wrap;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.project-configuration-workspace__tree-actions {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.project-configuration-workspace__tree-row:hover .project-configuration-workspace__tree-actions,
.project-configuration-workspace__tree-actions--visible {
  opacity: 1;
}

.project-configuration-workspace__annotation-button {
  color: var(--td-text-color-placeholder);
}

.project-configuration-workspace__annotation-button:hover {
  color: var(--td-brand-color-6);
}

.project-configuration-workspace__tree-error {
  color: var(--td-error-color-6);
  font: var(--td-font-body-small);
  margin: 0;
  padding-left: calc(
    (var(--workspace-tree-depth, 0) * var(--graft-density-gap-14)) + var(--graft-density-gap-24) +
      var(--graft-density-gap-2)
  );
}

.project-configuration-workspace__splitter {
  align-items: center;
  cursor: col-resize;
  display: flex;
  justify-content: center;
  min-height: 0;
  width: 2px;
}

.project-configuration-workspace__splitter-grip {
  background: color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: 999px;
  height: 40px;
  transition: background-color 0.2s ease;
  width: 1px;
}

.project-configuration-workspace__splitter:hover .project-configuration-workspace__splitter-grip {
  background: color-mix(in srgb, var(--td-brand-color-6) 55%, var(--td-component-stroke));
}

.project-configuration-workspace__editor-stack {
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__panel) {
  background: var(--graft-workspace-editor-surface);
  border-bottom: 0;
  border-color: var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-large) var(--td-radius-large) 0 0;
  box-shadow: 0 18px 34px color-mix(in srgb, var(--td-brand-color-6) 5%, transparent);
  display: flex;
  flex-direction: column;
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__header) {
  background: linear-gradient(180deg, color-mix(in srgb, var(--td-brand-color-6) 6%, transparent), transparent 72%);
  border-bottom-color: var(--graft-workspace-editor-border);
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__surface) {
  background: var(--graft-workspace-editor-surface);
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
}

.project-configuration-workspace__editor-stack :deep(.content-viewer-frame__resize-grip) {
  background: color-mix(in srgb, var(--td-text-color-placeholder) 42%, transparent);
}

.project-configuration-workspace__editor-surface {
  background: var(--graft-workspace-editor-surface);
  block-size: 100%;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-block-size: 0;
  min-inline-size: 0;
}

.project-configuration-workspace__tabs {
  margin-bottom: 0;

  :deep(.t-tabs__header) {
    background: linear-gradient(180deg, color-mix(in srgb, var(--td-brand-color-6) 4%, transparent), transparent);
    border-bottom: 1px solid var(--graft-workspace-editor-border);
    margin: 0;
    padding: 0 var(--graft-density-gap-8);
  }

  :deep(.t-tabs__nav) {
    align-items: stretch;
  }

  :deep(.t-tabs__bar) {
    display: none;
  }

  :deep(.t-tabs__nav-item) {
    background: transparent;
    border: 0;
    border-radius: 0;
    border-top: 2px solid transparent;
    color: var(--td-text-color-secondary);
    margin: 0;
    min-height: 42px;
    padding: 0 var(--graft-density-gap-12);
    transition:
      color 0.2s ease,
      background-color 0.2s ease,
      border-color 0.2s ease;
  }

  :deep(.t-tabs__nav-item:hover) {
    background: var(--graft-workspace-tab-hover);
    color: var(--td-text-color-primary);
  }

  :deep(.t-tabs__nav-item.t-is-active) {
    background: color-mix(in srgb, var(--td-brand-color-6) 6%, transparent);
    border-top-color: var(--graft-workspace-tab-indicator);
    color: var(--td-text-color-primary);
  }

  :deep(.t-tabs__nav-item + .t-tabs__nav-item) {
    box-shadow: inset 1px 0 0 color-mix(in srgb, var(--graft-workspace-editor-border) 72%, transparent);
  }
}

.project-configuration-workspace__tab-label {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-4);
}

.project-configuration-workspace__tab-dirty {
  color: var(--td-warning-color-6);
  font: var(--td-font-body-small);
  line-height: 1;
}

.project-configuration-workspace__editor-loading,
.project-configuration-workspace__monaco-editor,
.project-configuration-workspace__monaco-viewer {
  background: var(--graft-workspace-editor-surface);
  block-size: 100%;
  display: block;
  min-block-size: 0;
  min-inline-size: 0;
}

.project-configuration-workspace__editor-loading {
  display: flex;
  flex: 1 1 auto;
  min-block-size: 0;
  padding: 0 var(--graft-density-gap-8) var(--graft-density-gap-8);
}

.project-configuration-workspace__editor-stage {
  display: flex;
  flex: 1 1 auto;
  min-block-size: 0;
}

.project-configuration-workspace__editor-loading :deep(.t-loading__parent),
.project-configuration-workspace__editor-loading :deep(.t-loading__content),
.project-configuration-workspace__editor-loading :deep(.t-loading__wrap) {
  display: flex;
  flex: 1 1 auto;
  min-block-size: 0;
  min-inline-size: 0;
}

.project-configuration-workspace__monaco-editor,
.project-configuration-workspace__monaco-viewer {
  flex: 1 1 auto;
  overflow: hidden;
}

.project-configuration-workspace__monaco-editor :deep(.monaco-editor),
.project-configuration-workspace__monaco-editor :deep(.monaco-diff-editor),
.project-configuration-workspace__monaco-viewer :deep(.monaco-editor),
.project-configuration-workspace__monaco-viewer :deep(.monaco-diff-editor) {
  background: var(--graft-workspace-editor-surface);
}

.project-configuration-workspace__warning-list,
.project-configuration-workspace__feedback-panel {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-height: 0;
}

.project-configuration-workspace__diff-surface {
  display: grid;
  flex: 1 1 auto;
  gap: var(--graft-density-gap-10);
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr);
  min-height: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12) var(--graft-density-gap-12);
}

.project-configuration-workspace__diff-sidebar {
  background: var(--graft-workspace-editor-surface-muted);
  border: 1px solid var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-large);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-height: 0;
  min-width: 0;
  overflow: auto;
  padding: var(--graft-density-gap-8);
}

.project-configuration-workspace__diff-sidebar-head {
  border-bottom: 1px solid color-mix(in srgb, var(--graft-workspace-editor-border) 82%, transparent);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  margin-bottom: var(--graft-density-gap-6);
  padding-bottom: var(--graft-density-gap-8);
}

.project-configuration-workspace__diff-sidebar-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.project-configuration-workspace__diff-sidebar-caption {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-configuration-workspace__tree-row--diff {
  padding-right: var(--graft-density-gap-4);
}

.project-configuration-workspace__diff-stage {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-height: 0;
}

.project-configuration-workspace__diff-pane-heads {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-configuration-workspace__diff-pane-head {
  align-items: center;
  background: var(--graft-workspace-editor-surface-muted);
  border: 1px solid var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-default);
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-10);
}

.project-configuration-workspace__diff-pane-label {
  color: var(--td-text-color-secondary);
  flex: 0 0 auto;
  font: var(--td-font-body-small);
}

.project-configuration-workspace__hash-text {
  color: var(--td-text-color-primary);
  display: inline-block;
  flex: 1 1 auto;
  font-family: var(--td-font-family-mono, monospace);
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.project-configuration-workspace__result-dialog {
  background: var(--graft-workspace-editor-surface);
  block-size: 100%;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
}

.project-configuration-workspace__result-dialog-header {
  border-bottom: 1px solid var(--graft-workspace-editor-border);
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12) var(--graft-density-gap-8);
}

.project-configuration-workspace__result-dialog > .project-configuration-workspace__feedback-panel {
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12) var(--graft-density-gap-12);
}

.project-configuration-workspace__result-dialog-footer {
  border-top: 1px solid var(--graft-workspace-editor-border);
  display: flex;
  justify-content: flex-end;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-12) var(--graft-density-gap-10);
}

.project-configuration-workspace__result-viewer,
.project-configuration-workspace__readonly-viewer,
.project-configuration-workspace__drawer-viewer {
  background: var(--graft-workspace-editor-surface-muted);
  border: 1px solid var(--graft-workspace-editor-border);
  border-radius: var(--td-radius-large);
  display: flex;
  flex: 1 1 auto;
  min-block-size: 360px;
  min-inline-size: 0;
  overflow: hidden;
}

.project-configuration-workspace__diff-stage .project-configuration-workspace__result-viewer {
  min-block-size: 0;
}

.project-configuration-workspace__result-viewer :deep(.project-monaco-diff-surface),
.project-configuration-workspace__result-viewer :deep(.project-monaco-surface),
.project-configuration-workspace__readonly-viewer :deep(.project-monaco-surface),
.project-configuration-workspace__drawer-viewer :deep(.project-monaco-surface) {
  display: flex;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
  min-width: 0;
  width: 100%;
}

.project-configuration-workspace__result-viewer :deep(.project-monaco-diff-surface .monaco-diff-editor),
.project-configuration-workspace__result-viewer :deep(.project-monaco-diff-surface .editor),
.project-configuration-workspace__result-viewer :deep(.monaco-diff-editor),
.project-configuration-workspace__result-viewer :deep(.monaco-editor),
.project-configuration-workspace__result-viewer :deep(.overflow-guard),
.project-configuration-workspace__readonly-viewer :deep(.monaco-editor),
.project-configuration-workspace__readonly-viewer :deep(.overflow-guard),
.project-configuration-workspace__drawer-viewer :deep(.monaco-editor),
.project-configuration-workspace__drawer-viewer :deep(.overflow-guard) {
  height: 100% !important;
  min-height: 0;
  min-width: 0;
  width: 100% !important;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog__body) {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  padding: 0;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog__header) {
  padding: 0;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog__body-content) {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

:deep(.project-configuration-workspace__result-dialog-shell .t-dialog) {
  display: flex;
  flex-direction: column;
  height: 80vh;
  max-height: calc(100vh - 48px);
  max-width: min(80vw, 1600px);
  width: min(80vw, 1600px);
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body),
:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body-content),
:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body--fullscreen),
:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body--fullscreen--without-footer) {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 0;
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog) {
  border-radius: 0;
  height: 100vh;
  max-height: 100vh;
  max-width: none;
  width: 100vw;
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body) {
  height: 100%;
}

:deep(.project-configuration-workspace__result-dialog-shell--fullscreen .t-dialog__body-content) {
  height: 100%;
}

.project-configuration-workspace__dialog-body {
  margin: 0;
}

@media (width <= 1024px) {
  .project-configuration-workspace__main-grid,
  .project-configuration-workspace__diff-surface {
    grid-template-columns: minmax(0, 1fr);
  }

  .project-configuration-workspace__splitter {
    display: none;
  }
}
</style>
