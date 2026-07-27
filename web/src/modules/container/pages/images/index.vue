<template>
  <div class="docker-images-page" data-page-type="list-form-detail">
    <management-page-header
      compact
      title-key="container.images.title"
      :title="t('container.images.title')"
      description-key="container.images.description"
      :description="t('container.images.description')"
      :source="{ labelKey: 'container.images.eyebrow', fallback: t('container.images.eyebrow') }"
    >
      <template #actions>
        <t-space>
          <t-button variant="outline" :loading="cleanupLoading" @click="openCleanup">
            {{ t('container.images.actions.cleanup') }}
          </t-button>
          <t-button theme="primary" @click="pullDrawerVisible = true">{{
            t('container.images.actions.pull')
          }}</t-button>
        </t-space>
      </template>
      <template #compactActions>
        <t-space>
          <t-button theme="primary" @click="pullDrawerVisible = true">{{
            t('container.images.actions.pull')
          }}</t-button>
          <t-dropdown :options="compactHeaderActions" trigger="click" @click="handleCompactHeaderAction">
            <t-button variant="outline">{{ t('container.images.actions.more') }}</t-button>
          </t-dropdown>
        </t-space>
      </template>
    </management-page-header>

    <management-statistics-bar layout="chips" :items="metrics" :compact-items="compactMetrics" aria-live="polite" />

    <management-toolbar sticky-compact>
      <template #filters>
        <t-input
          v-model="keyword"
          class="management-list-search"
          clearable
          :placeholder="t('container.images.searchCompact')"
          @clear="clearKeyword"
          @enter="applyKeyword"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
      </template>
    </management-toolbar>

    <management-paged-table
      v-model:current="pagination.current"
      v-model:page-size="pagination.pageSize"
      :columns="columns"
      :column-sets="imageColumnSets"
      density-scope="viewport"
      entity-card-layout="compact"
      :rows="images"
      presentation="entity"
      :loading="query.isFetching.value"
      :total="total"
      :footer-summary="footerSummary"
      :empty-title="t('container.images.emptyTitle')"
      :empty-description="
        t(submittedKeyword ? 'container.images.filteredEmptyDescription' : 'container.images.emptyDescription')
      "
      :selected-row-keys="selectedRowKeys"
      @select-change="handleSelectChange"
    >
      <template #toolbar>
        <table-view-toolbar
          :refresh-label="t('container.images.actions.refresh')"
          :refresh-loading="query.isFetching.value"
          @refresh="refresh"
        />
      </template>
      <template v-if="selectedRowKeys.length" #batch>
        <management-batch-bar
          :selected-label="t('container.images.batch.selected', { count: selectedRowKeys.length })"
          :clear-label="t('container.images.batch.cancelSelection')"
          clear-test-id="docker-images-batch-clear"
          @clear="clearSelection"
        >
          <t-button
            v-if="canRemove"
            size="small"
            theme="danger"
            variant="outline"
            :loading="batchRemoving"
            @click="openBatchRemove"
          >
            {{ t('container.images.batch.remove') }}
          </t-button>
        </management-batch-bar>
      </template>
      <template #cards>
        <t-empty
          v-if="!images.length && !query.isFetching.value"
          :title="t('container.images.emptyTitle')"
          :description="
            t(submittedKeyword ? 'container.images.filteredEmptyDescription' : 'container.images.emptyDescription')
          "
        >
          <template #action>
            <t-button v-if="submittedKeyword" variant="outline" @click="clearKeyword">
              {{ t('container.images.clearFilter') }}
            </t-button>
          </template>
        </t-empty>
        <div v-else class="docker-images-card-list">
          <article
            v-for="image in images"
            :key="image.id"
            class="docker-images-card"
            :class="{ 'docker-images-card--selected': isImageSelected(image) }"
            :data-testid="`docker-image-card-${image.id}`"
            role="button"
            tabindex="0"
            @click="handleCardClick(image)"
            @keydown.enter.prevent="handleCardClick(image)"
            @keydown.space.prevent="handleCardClick(image)"
          >
            <header class="docker-images-card__header">
              <div class="docker-images-card__identity">
                <strong>{{ imageReference(imageTags(image)[0] ?? '').repository || shortId(image.id) }}</strong>
                <t-space v-if="imageTags(image).length" size="small" break-line>
                  <t-tag v-for="tag in imageTags(image)" :key="tag" size="small" variant="light-outline">
                    {{ imageReference(tag).tag }}
                  </t-tag>
                </t-space>
                <span v-else class="docker-images-muted">{{ t('container.images.untagged') }}</span>
              </div>
              <t-checkbox
                v-if="cardSelectionMode"
                :checked="isImageSelected(image)"
                :aria-label="t('container.images.selection.toggle', { image: shortId(image.id) })"
                @click.stop
                @change="toggleCardSelection(image, $event)"
              />
            </header>
            <div class="docker-images-card__primary">
              <strong>{{ formatBytes(image.size_bytes) }}</strong>
              <t-tag :theme="imageStatus(image).theme" size="small" variant="light-outline">
                {{ imageStatus(image).label }}
              </t-tag>
            </div>
            <dl class="docker-images-card__details">
              <div>
                <dt>{{ t('container.images.fields.containers') }}</dt>
                <dd>
                  <t-space v-if="image.container_references?.length" size="small" break-line>
                    <t-tag
                      v-for="container in image.container_references"
                      :key="container.id"
                      size="small"
                      variant="light-outline"
                    >
                      {{ container.name }}
                    </t-tag>
                  </t-space>
                  <span v-else>{{ t('container.images.unused') }}</span>
                </dd>
              </div>
              <div>
                <dt>{{ t('container.images.fields.createdAt') }}</dt>
                <dd>{{ formatLocaleDateTime(image.created_at, locale) }}</dd>
              </div>
            </dl>
            <footer class="docker-images-card__actions" @click.stop>
              <t-button size="small" variant="outline" @click="openDetail(image)">
                {{ t('container.images.actions.detail') }}
              </t-button>
              <t-dropdown
                :options="cardMoreActions"
                trigger="click"
                @click="handleCardDropdownAction($event.value, image)"
              >
                <t-button size="small" variant="outline">{{ t('container.images.actions.more') }}</t-button>
              </t-dropdown>
            </footer>
          </article>
        </div>
      </template>
      <template #feedback>
        <t-alert
          v-if="query.isError.value"
          theme="error"
          :message="resolveLocalizedErrorMessage(t, query.error.value, t('container.images.loadFailed'))"
        />
      </template>
      <template #empty>
        <t-empty
          :title="t('container.images.emptyTitle')"
          :description="
            t(submittedKeyword ? 'container.images.filteredEmptyDescription' : 'container.images.emptyDescription')
          "
        >
          <template #action>
            <t-button v-if="submittedKeyword" variant="outline" @click="clearKeyword">
              {{ t('container.images.clearFilter') }}
            </t-button>
          </template>
        </t-empty>
      </template>
      <template #tags="{ row }">
        <div class="docker-images-identity">
          <t-tooltip :content="imageReference(imageTags(row)[0] ?? '').repository">
            <strong class="docker-images-identity__repository">{{
              imageReference(imageTags(row)[0] ?? '').repository || shortId(row.id)
            }}</strong>
          </t-tooltip>
          <t-space size="small" break-line>
            <t-tooltip v-for="tag in imageTags(row).slice(0, 2)" :key="tag" :content="tag">
              <t-tag size="small" variant="light-outline">{{ imageReference(tag).tag }}</t-tag>
            </t-tooltip>
            <t-tag v-if="imageTags(row).length > 2" size="small" variant="light-outline"
              >+{{ imageTags(row).length - 2 }}</t-tag
            >
            <span v-if="!imageTags(row).length" class="docker-images-muted">{{ t('container.images.untagged') }}</span>
          </t-space>
        </div>
      </template>
      <template #status="{ row }">
        <t-tag :theme="imageStatus(row).theme" size="small" variant="light-outline">
          {{ imageStatus(row).label }}
        </t-tag>
      </template>
      <template #size="{ row }">{{ formatBytes(row.size_bytes) }}</template>
      <template #containers="{ row }">
        <t-space v-if="row.container_references?.length" size="small" break-line>
          <t-tooltip v-for="container in row.container_references" :key="container.id" :content="container.id">
            <t-tag size="small" variant="light-outline">{{ container.name }}</t-tag>
          </t-tooltip>
        </t-space>
        <t-tag v-else size="small" variant="light-outline">{{ t('container.images.unused') }}</t-tag>
      </template>
      <template #created_at="{ row }">{{ formatLocaleDateTime(row.created_at, locale) }}</template>
      <template #actions="{ row }">
        <table-action-menu
          :actions="[
            { value: 'detail', label: 'container.images.actions.detail' },
            { value: 'manage-tags', label: 'container.images.actions.manageTags' },
            { value: 'tag', label: 'container.images.actions.tag' },
            ...(canRemove ? [{ value: 'remove', label: 'container.images.actions.remove' }] : []),
          ]"
          :more-label="t('container.images.actions.more')"
          @action="handleRowAction($event, row)"
        />
      </template>
    </management-paged-table>

    <resource-detail-layout
      v-model:visible="detailDrawerVisible"
      :title="
        selectedImage
          ? imageReference(imageTags(selectedImage)[0] ?? '').repository || t('container.images.detail.title')
          : t('container.images.detail.title')
      "
      :back-label="t('container.detail.back')"
      size="medium"
    >
      <div class="docker-images-detail-loading-host">
        <div v-if="detailLoading" class="docker-images-detail-loading-host__indicator">
          <t-loading :loading="true" size="large" />
        </div>
        <div v-else-if="detailError" class="docker-images-detail-error">
          <t-alert theme="error" :message="detailError" />
        </div>
        <div v-else-if="selectedImage" class="docker-images-detail">
          <section class="docker-images-detail__section">
            <h3>{{ t('container.images.detail.overview') }}</h3>
            <div class="docker-images-detail__identity">
              <div>
                <span class="docker-images-detail__label">{{ t('container.images.fields.repository') }}</span>
                <t-tooltip
                  :content="imageReference(imageTags(selectedImage)[0] ?? '').repository || '-'"
                  placement="top-left"
                >
                  <strong>{{
                    middleEllipsis(imageReference(imageTags(selectedImage)[0] ?? '').repository || '-', 44)
                  }}</strong>
                </t-tooltip>
              </div>
              <t-tag :theme="imageStatus(selectedImage).theme" size="small" variant="light-outline">
                {{ imageStatus(selectedImage).label }}
              </t-tag>
            </div>
            <div class="docker-images-detail__value">
              <span class="docker-images-detail__label">{{ t('container.images.fields.tags') }}</span>
              <t-space v-if="imageTags(selectedImage).length" size="small" break-line>
                <t-tooltip v-for="tag in imageTags(selectedImage)" :key="tag" :content="tag">
                  <t-tag size="small" variant="light-outline">{{ imageReference(tag).tag }}</t-tag>
                </t-tooltip>
              </t-space>
              <span v-else>{{ t('container.images.untagged') }}</span>
            </div>
          </section>

          <section class="docker-images-detail__section">
            <h3>{{ t('container.images.detail.basicInfo') }}</h3>
            <dl class="docker-images-detail__grid">
              <div>
                <dt>{{ t('container.images.fields.size') }}</dt>
                <dd>{{ formatBytes(selectedImage.size_bytes) }}</dd>
              </div>
              <div>
                <dt>{{ t('container.images.fields.createdAt') }}</dt>
                <dd>{{ formatLocaleDateTime(selectedImage.created_at, locale) }}</dd>
              </div>
              <div>
                <dt>{{ t('container.images.fields.platform') }}</dt>
                <dd>{{ selectedImage.operating_system || '-' }} / {{ selectedImage.architecture || '-' }}</dd>
              </div>
              <div>
                <dt>{{ t('container.images.fields.containers') }}</dt>
                <dd>{{ selectedImage.container_references?.length || t('container.images.unused') }}</dd>
              </div>
            </dl>
          </section>

          <section class="docker-images-detail__section">
            <h3>{{ t('container.images.detail.metadata') }}</h3>
            <dl class="docker-images-detail__metadata">
              <div>
                <dt>{{ t('container.images.fields.id') }}</dt>
                <dd>
                  <t-tooltip :content="selectedImage.id"
                    ><span class="docker-images-code">{{ middleEllipsis(selectedImage.id) }}</span></t-tooltip
                  >
                </dd>
              </div>
              <div>
                <dt>{{ t('container.images.fields.digests') }}</dt>
                <dd v-if="selectedImage.repository_digests?.length">
                  <t-tooltip :content="(selectedImage.repository_digests ?? []).join(', ')"
                    ><span class="docker-images-code">{{
                      middleEllipsis((selectedImage.repository_digests ?? []).join(', '), 44)
                    }}</span></t-tooltip
                  >
                </dd>
                <dd v-else>-</dd>
              </div>
              <div>
                <dt>{{ t('container.images.fields.labels') }}</dt>
                <dd v-if="Object.keys(selectedImage.labels ?? {}).length">
                  <t-space direction="vertical" size="small">
                    <t-tooltip v-for="(value, key) in selectedImage.labels" :key="key" :content="`${key}=${value}`">
                      <span class="docker-images-code">{{ middleEllipsis(`${key}=${value}`, 44) }}</span>
                    </t-tooltip>
                  </t-space>
                </dd>
                <dd v-else>-</dd>
              </div>
            </dl>
          </section>
        </div>
        <div v-else-if="!detailLoading" class="docker-images-detail-state">
          <t-empty
            size="small"
            :title="t('container.images.detail.emptyTitle')"
            :description="t('container.images.detail.emptyDescription')"
          />
        </div>
        <t-alert
          v-if="selectedImage && imageTags(selectedImage).length > 1"
          class="docker-images-delete-preflight"
          theme="info"
          :message="t('container.images.remove.multipleTagsPreflight', { count: imageTags(selectedImage).length })"
        >
          <template #operation>
            <t-button size="small" variant="text" @click="openTagManager(selectedImage)">{{
              t('container.images.actions.manageTags')
            }}</t-button>
          </template>
        </t-alert>
        <container-danger-zone
          v-if="selectedImage && canRemove"
          :action-label="t('container.images.actions.remove')"
          :description="t('container.images.remove.risk')"
          @action="openRemove(selectedImage)"
        />
      </div>
      <template #footer>
        <t-space v-if="selectedImage">
          <t-button variant="outline" @click="openTagManager(selectedImage)">{{
            t('container.images.actions.manageTags')
          }}</t-button>
        </t-space>
      </template>
    </resource-detail-layout>

    <tag-manager-drawer
      :visible="tagManagerVisible"
      :image-id="tagManagerImageId"
      @update:visible="handleTagManagerVisibleChange"
      @refreshed="handleTagManagerRefreshed"
    />

    <t-dialog
      v-model:visible="tagDialogVisible"
      :header="t('container.images.tag.title')"
      :confirm-btn="t('container.images.actions.tag')"
      :confirm-loading="tagging"
      @confirm="submitTag"
    >
      <t-form label-align="top">
        <t-form-item :label="t('container.images.tag.repository')"
          ><t-input v-model="tagForm.repository"
        /></t-form-item>
        <t-form-item :label="t('container.images.tag.tag')"><t-input v-model="tagForm.tag" /></t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="multiTagFailureDialogVisible"
      theme="warning"
      :header="t('container.images.remove.multipleTagsTitle')"
      :confirm-btn="t('container.images.actions.manageTags')"
      @confirm="openFailedImageTagManager"
    >
      <p>{{ t('container.images.remove.multipleTagsFailed') }}</p>
    </t-dialog>

    <t-dialog
      v-model:visible="removeDialogVisible"
      theme="danger"
      :header="t('container.images.remove.title')"
      :confirm-btn="removeConfirmButton"
      :confirm-loading="removing || batchRemoving"
      @confirm="submitRemove"
    >
      <p v-if="selectedImage">{{ t('container.images.remove.confirm', { image: shortId(selectedImage.id) }) }}</p>
      <template v-else>
        <p>{{ t('container.images.batch.confirm', { count: selectedRowKeys.length }) }}</p>
        <p class="docker-images-risk">{{ t('container.images.batch.riskIntro') }}</p>
        <ul class="docker-images-risk-list">
          <li>{{ t('container.images.batch.riskMultipleTags') }}</li>
          <li>{{ t('container.images.batch.riskContainerReference') }}</li>
        </ul>
        <p class="docker-images-risk">{{ t('container.images.batch.normalRemovalOnly') }}</p>
        <template v-if="selectedBatchReferences.length">
          <p class="docker-images-risk">{{ t('container.images.batch.inUse') }}</p>
          <t-space size="small" break-line>
            <t-tooltip v-for="container in selectedBatchReferences" :key="container.id" :content="container.id">
              <t-tag size="small" variant="light-outline">{{ container.name }}</t-tag>
            </t-tooltip>
          </t-space>
          <t-checkbox v-model="forceRemove">{{ t('container.images.remove.force') }}</t-checkbox>
        </template>
      </template>
      <p v-if="selectedImage?.container_references?.length" class="docker-images-risk">
        {{ t('container.images.remove.inUse') }}
      </p>
      <t-checkbox v-if="selectedImage?.container_references?.length" v-model="forceRemove">{{
        t('container.images.remove.force')
      }}</t-checkbox>
    </t-dialog>

    <t-dialog
      v-model:visible="batchResultDialogVisible"
      dialog-class-name="docker-images-result-dialog"
      :header="t('container.images.batch.resultTitle')"
      width="640px"
      :confirm-btn="t('container.images.batch.confirmDetails')"
      :cancel-btn="null"
      @confirm="batchResultDialogVisible = false"
    >
      <template v-if="batchResultUnknown">
        <p class="docker-images-result__global-message">{{ t('container.images.batch.resultUnknown') }}</p>
      </template>
      <template v-else>
        <div class="docker-images-result-summary">
          <div>
            <span>{{ t('container.images.batch.resultSuccess') }}</span
            ><strong>{{ batchResultSuccessCount }}</strong>
          </div>
          <div>
            <span>{{ t('container.images.batch.resultFailure') }}</span
            ><strong>{{ batchFailureDetails.length }}</strong>
          </div>
        </div>
        <div class="docker-images-failure-list">
          <section v-for="group in batchFailureGroups" :key="group.code" class="docker-images-failure-group">
            <header class="docker-images-failure-group__header">
              <div>
                <h3>{{ group.title }}</h3>
                <p>{{ group.description }}</p>
              </div>
              <t-tag theme="danger" variant="light-outline">{{ group.items.length }}</t-tag>
            </header>
            <div v-for="failure in group.items" :key="failure.id" class="docker-images-failure-item">
              <div class="docker-images-failure-item__header">
                <div class="docker-images-failure-item__identity">
                  <t-tag theme="danger" variant="light-outline" size="small">{{ shortId(failure.id) }}</t-tag>
                  <span class="docker-images-failure-item__name">{{ failure.name }}</span>
                </div>
                <t-button
                  v-if="failure.code === DOCKER_IMAGE_REMOVE_ERROR_CODES.IMAGE_REFERENCED_BY_MULTIPLE_TAGS"
                  class="docker-images-failure-item__manage-tags"
                  size="small"
                  variant="outline"
                  @click="openBatchFailureTagManager(failure.id)"
                >
                  {{ t('container.images.actions.manageTags') }}
                </t-button>
              </div>
              <t-collapse v-if="failure.tags.length" borderless expand-icon-placement="right">
                <t-collapse-panel
                  :value="`tags-${failure.id}`"
                  :header="t('container.images.batch.tagCount', { count: failure.tags.length })"
                >
                  <t-space size="small" break-line>
                    <t-tag v-for="tag in failure.tags" :key="tag" size="small" variant="light-outline">{{ tag }}</t-tag>
                  </t-space>
                </t-collapse-panel>
              </t-collapse>
            </div>
          </section>
        </div>
      </template>
    </t-dialog>

    <t-dialog
      v-model:visible="cleanupDialogVisible"
      dialog-class-name="docker-images-cleanup-dialog"
      :dialog-style="cleanupDialogStyle"
      :header="false"
      width="760px"
      @confirm="submitCleanup"
    >
      <div class="docker-images-cleanup graft-scrollbar">
        <header class="docker-images-cleanup__header">
          <div class="docker-images-cleanup__header-icon"><delete-icon /></div>
          <div>
            <h2>{{ t('container.images.cleanup.title') }}</h2>
            <p>{{ t('container.images.cleanup.subtitle') }}</p>
          </div>
        </header>

        <t-loading :loading="cleanupLoading">
          <t-card v-if="cleanupImages.length" class="docker-images-cleanup-summary" :bordered="false">
            <div class="docker-images-cleanup-summary__stats">
              <div>
                <span class="docker-images-cleanup-summary__label">{{
                  t('container.images.cleanup.candidateCount')
                }}</span>
                <strong>{{ cleanupImages.length }}</strong>
                <span>{{ t('container.images.cleanup.imageUnit') }}</span>
              </div>
              <div>
                <span class="docker-images-cleanup-summary__label">{{
                  t('container.images.cleanup.releaseSize')
                }}</span>
                <strong>{{ formatBytes(cleanupTotalSize) }}</strong>
              </div>
              <div>
                <span class="docker-images-cleanup-summary__label">{{ t('container.images.cleanup.source') }}</span>
                <strong>{{ t('container.images.cleanup.sourceValue') }}</strong>
              </div>
            </div>
          </t-card>

          <t-alert
            v-if="cleanupImages.length"
            class="docker-images-cleanup-warning"
            theme="warning"
            :message="t('container.images.cleanup.warning')"
          />

          <section v-if="cleanupImages.length" class="docker-images-cleanup-preview">
            <div class="docker-images-cleanup-section-head">
              <h3>{{ t('container.images.cleanup.candidateTitle', { count: cleanupImages.length }) }}</h3>
              <t-space size="small">
                <span>{{ t('container.images.cleanup.selectedCount', { count: cleanupSelectedIds.length }) }}</span>
                <t-button v-if="cleanupSelectedIds.length" size="small" variant="text" @click="clearCleanupSelection">
                  {{ t('container.images.cleanup.clearSelection') }}
                </t-button>
              </t-space>
            </div>

            <div class="docker-images-cleanup-preview-body">
              <t-table
                class="docker-images-cleanup-table"
                :columns="cleanupColumns"
                :data="cleanupPreviewImages"
                row-key="id"
                size="small"
                table-layout="fixed"
                :selected-row-keys="cleanupSelectedIds"
                @select-change="handleCleanupSelectChange"
              >
                <template #image="{ row }">
                  <div class="docker-images-cleanup-image">
                    <image-icon class="docker-images-cleanup-image__icon" />
                    <span class="docker-images-cleanup-image__name">{{
                      imageTags(row).join(', ') || shortId(row.id)
                    }}</span>
                  </div>
                </template>
                <template #status>
                  <t-tag size="small" variant="light-outline">{{ t('container.images.unused') }}</t-tag>
                </template>
                <template #size="{ row }">{{ formatBytes(row.size_bytes) }}</template>
              </t-table>

              <div v-if="cleanupPreviewPageCount > 1" class="docker-images-cleanup-pager">
                <t-tooltip :content="t('container.images.cleanup.previousPage')">
                  <t-button
                    size="small"
                    variant="text"
                    shape="circle"
                    :disabled="cleanupPreviewPage === 1"
                    :aria-label="t('container.images.cleanup.previousPage')"
                    @click="previousCleanupPage"
                  >
                    <arrow-up-icon />
                  </t-button>
                </t-tooltip>
                <span>{{ cleanupPreviewPage }} / {{ cleanupPreviewPageCount }}</span>
                <t-tooltip :content="t('container.images.cleanup.nextPage')">
                  <t-button
                    size="small"
                    variant="text"
                    shape="circle"
                    :disabled="cleanupPreviewPage === cleanupPreviewPageCount"
                    :aria-label="t('container.images.cleanup.nextPage')"
                    @click="nextCleanupPage"
                  >
                    <arrow-down-icon />
                  </t-button>
                </t-tooltip>
              </div>
            </div>
          </section>
        </t-loading>

        <t-empty v-if="!cleanupLoading && !cleanupImages.length" :title="t('container.images.cleanup.empty')" />
      </div>
      <template #footer>
        <div class="docker-images-cleanup-footer">
          <div>
            <span>{{ t('container.images.cleanup.footerRelease') }}</span>
            <strong>{{ formatBytes(cleanupSelectedSize) }}</strong>
          </div>
          <t-space size="small">
            <t-button variant="outline" @click="cleanupDialogVisible = false">
              {{ t('container.images.cleanup.cancel') }}
            </t-button>
            <t-button
              theme="danger"
              :disabled="!cleanupSelectedIds.length"
              :loading="batchRemoving"
              @click="submitCleanup"
            >
              {{ t('container.images.cleanup.removeSelected', { count: cleanupSelectedIds.length }) }}
            </t-button>
          </t-space>
        </div>
      </template>
    </t-dialog>

    <t-drawer
      v-model:visible="pullDrawerVisible"
      :header="t('container.images.pull.title')"
      size="720px"
      placement="right"
    >
      <t-form label-align="top" @submit.prevent="startPull">
        <t-form-item :label="t('container.images.pull.reference')">
          <t-input v-model="pullReference" :disabled="pulling" :placeholder="t('container.images.pull.placeholder')" />
        </t-form-item>
        <t-space>
          <t-button theme="primary" :loading="pulling" @click="startPull">{{
            t('container.images.actions.pull')
          }}</t-button>
        </t-space>
      </t-form>
    </t-drawer>

    <task-detail-drawer
      :visible="pullTaskDrawerVisible"
      :task-id="pullTaskId"
      :resolve-task-type="resolveTaskType"
      @update:visible="pullTaskDrawerVisible = $event"
    />
  </div>
</template>
<script setup lang="ts">
// 镜像页面只管理当前 Docker runtime 的镜像快照；批量删除按服务端上限分块，结构化拒绝直接展示稳定错误，网络结果未知时才通过详情查询对账。
import { ArrowDownIcon, ArrowUpIcon, DeleteIcon, ImageIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onUnmounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  CONTAINER_PERMISSION_CODE,
  DOCKER_IMAGE_REMOVE_ERROR_CODES,
  type DockerImageRemoveErrorCode,
} from '@/contracts/generated/modules/container';
import {
  ManagementBatchBar,
  ManagementPagedTable,
  ManagementPageHeader,
  ManagementStatisticsBar,
  ManagementToolbar,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import ResourceDetailLayout from '@/shared/components/responsive/ResourceDetailLayout.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatBytes, formatLocaleDateTime } from '@/shared/observability';
import { usePermissionStore } from '@/store';
import { createLogger } from '@/utils/logger';
import { isApiRequestError } from '@/utils/request';

import { isTerminalTaskStatus, observeTask, type TaskObserver } from '../../../task/contract/task-observer';
import { TaskDetailDrawer } from '../../../task/contract/task-ui';
import { type DockerImageRecord, getDockerImage, getDockerImages } from '../../api/container';
import {
  batchRemoveDockerImages,
  type DockerImageBatchResult,
  pullDockerImage,
  removeDockerImage,
  tagDockerImage,
} from '../../api/image-actions';
import ContainerDangerZone from '../../components/ContainerDangerZone.vue';
import TagManagerDrawer from '../../components/TagManagerDrawer.vue';
import { CONTAINER_TASK_TYPE } from '../../contract/task-types';
import { useDockerCleanup } from '../../shared/cleanup/use-docker-cleanup';
import { type DockerImageQueryState, useDockerImageQuery } from '../../shared/docker-image-queries';

type DockerImage = DockerImageRecord;
type BatchFailureDetail = { id: string; name: string; tags: string[]; code: DockerImageRemoveErrorCode };
type BatchFailureGroup = {
  code: DockerImageRemoveErrorCode;
  title: string;
  description: string;
  items: BatchFailureDetail[];
};

const dockerImageReferencedByMultipleTagsMessageKey = 'ops.container.error.imageReferencedByMultipleTags';
let pullIdempotencySequence = 0;

const { locale, t } = useI18n();
const logger = createLogger('container.images');
const permissionStore = usePermissionStore();
const pagination = reactive({ current: 1, pageSize: 20 });
const keyword = ref('');
const submittedKeyword = ref('');
const imageQuery = computed<DockerImageQueryState>(() => ({
  pageSize: pagination.pageSize,
  offset: (pagination.current - 1) * pagination.pageSize,
  keyword: submittedKeyword.value,
}));
const query = useDockerImageQuery(imageQuery);
const selectedImage = ref<DockerImage | null>(null);
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const detailError = ref('');
const tagDialogVisible = ref(false);
const tagManagerVisible = ref(false);
const tagManagerImageId = ref<string | null>(null);
const restoreBatchResultAfterTagManager = ref(false);
const multiTagFailureDialogVisible = ref(false);
const failedMultiTagImageId = ref<string | null>(null);
const removeDialogVisible = ref(false);
const batchResultDialogVisible = ref(false);
const batchFailureDetails = ref<BatchFailureDetail[]>([]);
const batchResultSuccessCount = ref(0);
const batchResultUnknown = ref(false);
const pullDrawerVisible = ref(false);
const pullTaskDrawerVisible = ref(false);
const pullTaskId = ref<number | null>(null);
const tagging = ref(false);
const removing = ref(false);
const forceRemove = ref(false);
const selectedRowKeys = ref<Array<string | number>>([]);
const selectedImages = ref(new Map<string, DockerImage>());
const canRemove = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.IMAGE_REMOVE));
const cardSelectionMode = ref(false);
const batchRemoving = ref(false);
const cleanupDialogStyle = { maxHeight: '70vh' };
const tagForm = reactive({ repository: '', tag: '' });
const pullReference = ref('');
const pulling = ref(false);
// 拉取进度由 Task Runtime 统一呈现，页面只保留可在卸载时释放的状态观察订阅。
let pullTaskObserver: TaskObserver | null = null;
const cleanup = useDockerCleanup<DockerImage>({
  fetchCandidates: fetchCleanupCandidates,
  execute: (ids) => removeImageIds(ids, false),
});
const cleanupDialogVisible = cleanup.visible;
const cleanupLoading = cleanup.loading;
const cleanupImages = cleanup.items;
const cleanupSelectedIds = cleanup.selectedIds;
const cleanupPreviewPage = cleanup.previewPage;
const tagTarget = computed(() => {
  const repository = tagForm.repository.trim();
  const tag = tagForm.tag.trim();
  return repository && tag ? `${repository}:${tag}` : '';
});
const images = computed(() => query.data.value?.items ?? []);
const total = computed(() => query.data.value?.total ?? 0);
const summary = computed(() => query.data.value?.summary);
const metrics = computed(() => [
  { label: t('container.images.metrics.total'), value: summary.value ? summary.value.total : '--' },
  {
    label: t('container.images.metrics.size'),
    value: summary.value ? formatBytes(summary.value.size_bytes) : '--',
  },
  { label: t('container.images.metrics.inUse'), value: summary.value ? summary.value.in_use : '--' },
  { label: t('container.images.metrics.dangling'), value: summary.value ? summary.value.dangling : '--' },
]);
const compactMetrics = computed(() => [
  { label: t('container.images.metrics.totalCompact'), value: summary.value ? summary.value.total : '--' },
  {
    label: t('container.images.metrics.sizeCompact'),
    value: summary.value ? formatBytes(summary.value.size_bytes) : '--',
  },
  { label: t('container.images.metrics.inUseCompact'), value: summary.value ? summary.value.in_use : '--' },
]);
const compactHeaderActions = computed(() => [
  { content: t('container.images.actions.cleanup'), theme: 'error' as const, value: 'cleanup' },
]);
const footerSummary = computed(() => {
  if (!total.value) return t('container.images.pagination.empty');
  const start = (pagination.current - 1) * pagination.pageSize + 1;
  const end = Math.min(pagination.current * pagination.pageSize, total.value);
  return t('container.images.pagination.summary', { start, end, total: total.value });
});
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'tags', title: t('container.images.fields.name'), ellipsis: true, minWidth: 280 },
  { colKey: 'size', title: t('container.images.fields.size'), width: 120 },
  { colKey: 'containers', title: t('container.images.fields.containers'), minWidth: 220 },
  { colKey: 'status', title: t('container.images.fields.status'), width: 110 },
  { colKey: 'created_at', title: t('container.images.fields.createdAt'), width: 180 },
  { colKey: 'actions', title: t('container.images.fields.actions'), width: 150, fixed: 'right' as const },
]);
const imageColumnSets = computed(() => ({
  comfortable: ['tags', 'size', 'status', 'actions'],
  spacious: ['row-select', 'tags', 'size', 'containers', 'status', 'created_at', 'actions'],
}));
const cardMoreActions = computed(() => [
  { content: t('container.images.actions.select'), value: 'select' },
  { content: t('container.images.actions.manageTags'), value: 'manage-tags' },
  { content: t('container.images.actions.tag'), value: 'tag' },
  ...(canRemove.value
    ? [{ content: t('container.images.actions.remove'), theme: 'error' as const, value: 'remove' }]
    : []),
]);
const selectedBatchReferences = computed(() =>
  selectedRowKeys.value.flatMap((key) => selectedImages.value.get(String(key))?.container_references ?? []),
);
const removeConfirmButton = computed(() => ({
  content: t('container.images.actions.remove'),
  disabled:
    (!selectedImage.value && !selectedRowKeys.value.length) ||
    (!forceRemove.value &&
      (Boolean(selectedImage.value?.container_references?.length) || Boolean(selectedBatchReferences.value.length))),
}));
const batchFailureGroups = computed<BatchFailureGroup[]>(() => {
  const groups = new Map<DockerImageRemoveErrorCode, BatchFailureDetail[]>();
  batchFailureDetails.value.forEach((item) => {
    groups.set(item.code, [...(groups.get(item.code) ?? []), item]);
  });
  return [...groups].map(([code, items]) => ({
    code,
    items,
    title: t(`container.images.batch.error.${code}.title`),
    description: t(`container.images.batch.error.${code}.description`),
  }));
});
const cleanupSelectedSize = cleanup.selectedSize;
const cleanupTotalSize = cleanup.totalSize;
const cleanupPreviewPageCount = cleanup.pageCount;
const cleanupPreviewImages = cleanup.previewItems;
const cleanupColumns = computed<TableProps['columns']>(() => [
  { colKey: 'row-select', type: 'multiple' as const, width: 48 },
  { colKey: 'image', title: t('container.images.cleanup.imageColumn'), ellipsis: true, minWidth: 280 },
  { colKey: 'status', title: t('container.images.cleanup.statusColumn'), width: 92 },
  { colKey: 'size', title: t('container.images.cleanup.sizeColumn'), width: 112, align: 'right' as const },
]);

function imageTags(image: DockerImage) {
  return image.repository_tags?.filter(Boolean) ?? [];
}
function imageReference(reference: string) {
  const lastSlash = reference.lastIndexOf('/');
  const lastColon = reference.lastIndexOf(':');
  return lastColon > lastSlash
    ? { repository: reference.slice(0, lastColon), tag: reference.slice(lastColon + 1) }
    : { repository: reference, tag: '' };
}
function imageStatus(image: DockerImage) {
  if (image.container_references?.length)
    return { label: t('container.images.status.used'), theme: 'success' as const };
  if (image.dangling) return { label: t('container.images.status.dangling'), theme: 'warning' as const };
  return { label: t('container.images.status.unused'), theme: 'default' as const };
}
function middleEllipsis(value: string, maxLength = 28) {
  if (value.length <= maxLength) return value;
  const sideLength = Math.max(4, Math.floor((maxLength - 3) / 2));
  return `${value.slice(0, sideLength)}...${value.slice(-sideLength)}`;
}
function shortId(value: string) {
  return value.replace(/^sha256:/, '').slice(0, 12);
}
function refresh() {
  return query.refetch();
}
function applyKeyword() {
  submittedKeyword.value = keyword.value.trim();
  pagination.current = 1;
}
function clearKeyword() {
  keyword.value = '';
  submittedKeyword.value = '';
  pagination.current = 1;
}
function handleSelectChange(rowKeys: Array<string | number>) {
  images.value.forEach((image) => selectedImages.value.set(image.id, image));
  const currentPageIds = new Set(images.value.map((image) => image.id));
  const preserved = selectedRowKeys.value.filter((key) => !currentPageIds.has(String(key)));
  selectedRowKeys.value = [...preserved, ...rowKeys.filter((key) => currentPageIds.has(String(key)))];
}
function clearSelection() {
  selectedRowKeys.value = [];
  selectedImages.value.clear();
  cardSelectionMode.value = false;
}
function handleRowAction(action: string, image: DockerImage) {
  if (action === 'detail') openDetail(image);
  if (action === 'manage-tags') openTagManager(image);
  if (action === 'tag') openTag(image);
  if (action === 'remove' && canRemove.value) openRemove(image);
}
function handleCompactHeaderAction(action: { value?: string | number | Record<string, unknown> }) {
  if (action.value === 'cleanup') void openCleanup();
}
function handleCardAction(action: string, image: DockerImage) {
  if (action === 'select') {
    cardSelectionMode.value = true;
    setCardSelected(image, true);
    return;
  }
  handleRowAction(action, image);
}
function handleCardDropdownAction(action: string | number | Record<string, unknown> | undefined, image: DockerImage) {
  if (typeof action === 'string') handleCardAction(action, image);
}
function handleCardClick(image: DockerImage) {
  if (cardSelectionMode.value) {
    setCardSelected(image, !isImageSelected(image));
    return;
  }
  void openDetail(image);
}
function isImageSelected(image: DockerImage) {
  return selectedRowKeys.value.includes(image.id);
}
function toggleCardSelection(image: DockerImage, selected: boolean) {
  setCardSelected(image, selected);
}
function setCardSelected(image: DockerImage, selected: boolean) {
  selectedImages.value.set(image.id, image);
  selectedRowKeys.value = selected
    ? [...new Set([...selectedRowKeys.value, image.id])]
    : selectedRowKeys.value.filter((key) => String(key) !== image.id);
}
function openTagManager(image: DockerImage) {
  restoreBatchResultAfterTagManager.value = false;
  tagManagerImageId.value = image.id;
  tagManagerVisible.value = true;
}
function handleTagManagerVisibleChange(visible: boolean) {
  tagManagerVisible.value = visible;
  if (visible || !restoreBatchResultAfterTagManager.value) return;
  restoreBatchResultAfterTagManager.value = false;
  batchResultDialogVisible.value = true;
}
async function handleTagManagerRefreshed(image: DockerImage | null) {
  if (image && selectedImage.value?.id === image.id) selectedImage.value = image;
  await refresh();
}
async function openDetail(image: DockerImage) {
  selectedImage.value = null;
  detailError.value = '';
  detailDrawerVisible.value = true;
  detailLoading.value = true;
  try {
    selectedImage.value = await getDockerImage(image.id);
  } catch (error) {
    detailError.value = resolveLocalizedErrorMessage(t, error, t('container.images.detail.loadFailed'));
  } finally {
    detailLoading.value = false;
  }
}
function openTag(image: DockerImage) {
  selectedImage.value = image;
  const reference = imageTags(image)[0] ?? '';
  const lastSlash = reference.lastIndexOf('/');
  const lastColon = reference.lastIndexOf(':');
  tagForm.repository = lastColon > lastSlash ? reference.slice(0, lastColon) : reference;
  tagForm.tag = 'latest';
  tagDialogVisible.value = true;
}
async function submitTag() {
  if (!selectedImage.value || !tagTarget.value) return;
  tagging.value = true;
  try {
    await tagDockerImage(selectedImage.value.id, { target: tagTarget.value });
    tagDialogVisible.value = false;
    await refresh();
    MessagePlugin.success(t('container.images.tag.success'));
  } catch {
    MessagePlugin.error(t('container.images.tag.failed'));
  } finally {
    tagging.value = false;
  }
}
function openRemove(image: DockerImage) {
  if (!canRemove.value) return;
  selectedImage.value = image;
  forceRemove.value = false;
  removeDialogVisible.value = true;
}
async function submitRemove() {
  if (!selectedImage.value) {
    await submitBatchRemove(selectedRowKeys.value.map(String), forceRemove.value);
    return;
  }
  if (selectedImage.value.container_references?.length && !forceRemove.value) return;
  removing.value = true;
  try {
    await removeDockerImage(selectedImage.value.id, { force: forceRemove.value });
    removeDialogVisible.value = false;
    forgetSelectedImages([selectedImage.value.id]);
    await refresh();
    MessagePlugin.success(t('container.images.remove.success'));
  } catch (error) {
    if (isMultipleTagFailure(error)) {
      failedMultiTagImageId.value = selectedImage.value.id;
      removeDialogVisible.value = false;
      multiTagFailureDialogVisible.value = true;
      return;
    }
    MessagePlugin.error(t('container.images.remove.failed'));
  } finally {
    removing.value = false;
  }
}
function openBatchRemove() {
  if (!selectedRowKeys.value.length) return;
  forceRemove.value = false;
  selectedImage.value = null;
  removeDialogVisible.value = true;
}
function forgetSelectedImages(ids: string[]) {
  const removedIds = new Set(ids);
  selectedRowKeys.value = selectedRowKeys.value.filter((key) => !removedIds.has(String(key)));
  ids.forEach((id) => selectedImages.value.delete(id));
}
async function removeImageIds(ids: string[], force = false) {
  const results: DockerImageBatchResult['items'] = [];
  const unknownResponseIds: string[] = [];
  let requestError: unknown;
  for (let index = 0; index < ids.length; index += 100) {
    const chunkIds = ids.slice(index, index + 100);
    try {
      const response = await batchRemoveDockerImages({ ids: chunkIds, force });
      results.push(...response.items);
    } catch (error) {
      logBatchRequestError('batch image removal request failed', error);
      if (isApiRequestError(error) && error.status > 0) {
        requestError = error;
        break;
      }
      unknownResponseIds.push(...chunkIds);
      results.push(
        ...chunkIds.map((id) => ({ id, success: false, error_code: DOCKER_IMAGE_REMOVE_ERROR_CODES.UNKNOWN })),
      );
    }
  }
  return { items: results, unknownResponseIds, requestError };
}
async function submitCleanup() {
  if (!cleanupSelectedIds.value.length) return;
  await submitBatchRemove(cleanupSelectedIds.value, false, true);
}
async function submitBatchRemove(ids: string[], force: boolean, cleanup = false) {
  batchRemoving.value = true;
  try {
    const { items, unknownResponseIds, requestError } = await removeImageIds(ids, force);
    const hasUnknownResponse = unknownResponseIds.length > 0;
    const successfulIds = new Set(items.filter((item) => item.success).map((item) => item.id));
    forgetSelectedImages([...successfulIds]);
    if (cleanup) {
      cleanupSelectedIds.value = cleanupSelectedIds.value.filter((id) => !successfulIds.has(id));
      cleanupImages.value = cleanupImages.value.filter((image) => !successfulIds.has(image.id));
      cleanupPreviewPage.value = Math.min(cleanupPreviewPage.value, cleanupPreviewPageCount.value);
      if (hasUnknownResponse) await reconcileCleanupCandidates(successfulIds);
    } else if (hasUnknownResponse) {
      await reconcileSelectedImages(unknownResponseIds);
    }
    const failed = items.filter((item) => !item.success);
    if (requestError || hasUnknownResponse) {
      closeBatchDialogs();
      MessagePlugin.error(t('container.images.batch.requestFailed'));
      showUnknownBatchResult();
    } else if (!failed.length) {
      MessagePlugin.success(t('container.images.batch.success', { count: items.length }));
    } else {
      if (failed.length < items.length) {
        MessagePlugin.warning(
          t('container.images.batch.partial', { success: items.length - failed.length, failed: failed.length }),
        );
      } else {
        MessagePlugin.error(t('container.images.batch.failed', { count: failed.length }));
      }
      closeBatchDialogs();
      showBatchFailureDetails(failed, items.length - failed.length);
    }
    if (!requestError && !hasUnknownResponse && (!failed.length || failed.length < items.length)) {
      removeDialogVisible.value = false;
      if (!cleanup) cleanupDialogVisible.value = false;
    }
    await refresh();
  } catch (error) {
    logBatchRequestError('batch image removal flow failed', error);
    MessagePlugin.error(t('container.images.batch.requestFailed'));
    closeBatchDialogs();
    showUnknownBatchResult();
  } finally {
    batchRemoving.value = false;
  }
}
function closeBatchDialogs() {
  removeDialogVisible.value = false;
  cleanupDialogVisible.value = false;
}

function showBatchFailureDetails(items: DockerImageBatchResult['items'], successCount: number) {
  batchResultUnknown.value = false;
  batchResultSuccessCount.value = successCount;
  batchFailureDetails.value = items.map((item) => {
    const image =
      selectedImages.value.get(item.id) ?? cleanupImages.value.find((candidate) => candidate.id === item.id);
    const tags = image ? imageTags(image) : [];
    return { id: item.id, name: tags[0] ?? shortId(item.id), tags, code: normalizeBatchFailureCode(item.error_code) };
  });
  batchResultDialogVisible.value = true;
}

function showUnknownBatchResult() {
  batchResultUnknown.value = true;
  batchResultSuccessCount.value = 0;
  batchFailureDetails.value = [];
  batchResultDialogVisible.value = true;
}

function logBatchRequestError(message: string, error: unknown) {
  if (isApiRequestError(error)) {
    logger.error(error, {
      operation: message,
      status: error.status,
      code: error.code,
      messageKey: error.messageKey,
      traceId: error.traceId,
      responseData: error.responseData,
    });
    return;
  }
  logger.error(error instanceof Error ? error : new Error(String(error)), { message });
}

function normalizeBatchFailureCode(errorCode?: string): DockerImageRemoveErrorCode {
  return Object.values(DOCKER_IMAGE_REMOVE_ERROR_CODES).includes(errorCode as DockerImageRemoveErrorCode)
    ? (errorCode as DockerImageRemoveErrorCode)
    : DOCKER_IMAGE_REMOVE_ERROR_CODES.UNKNOWN;
}

function isMultipleTagFailure(error: unknown) {
  return isApiRequestError(error) && error.messageKey === dockerImageReferencedByMultipleTagsMessageKey;
}
function openFailedImageTagManager() {
  if (!failedMultiTagImageId.value) return;
  restoreBatchResultAfterTagManager.value = false;
  tagManagerImageId.value = failedMultiTagImageId.value;
  tagManagerVisible.value = true;
  multiTagFailureDialogVisible.value = false;
}
function openBatchFailureTagManager(imageId: string) {
  restoreBatchResultAfterTagManager.value = true;
  tagManagerImageId.value = imageId;
  tagManagerVisible.value = true;
  batchResultDialogVisible.value = false;
}
async function openCleanup() {
  try {
    await cleanup.open();
  } catch {
    MessagePlugin.error(t('container.images.cleanup.loadFailed'));
  }
}
async function fetchCleanupCandidates() {
  const firstPage = await getDockerImages({ limit: 100, offset: 0, unused: true });
  const all = [...firstPage.items];
  for (let offset = firstPage.items.length; offset < firstPage.total; offset += 100) {
    const page = await getDockerImages({ limit: 100, offset, unused: true });
    all.push(...page.items);
  }
  return all;
}
async function reconcileCleanupCandidates(confirmedSuccessfulIds: Set<string>) {
  try {
    await cleanup.reconcile(confirmedSuccessfulIds);
  } catch {
    MessagePlugin.error(t('container.images.cleanup.loadFailed'));
  }
}
async function reconcileSelectedImages(ids: string[]) {
  const removedIds: string[] = [];
  await Promise.all(
    ids.map(async (id) => {
      try {
        selectedImages.value.set(id, await getDockerImage(id));
      } catch (error) {
        if (isApiRequestError(error) && error.status === 404) removedIds.push(id);
      }
    }),
  );
  selectedImages.value = new Map(selectedImages.value);
  forgetSelectedImages(removedIds);
}
function clearCleanupSelection() {
  cleanup.clearSelection();
}
function handleCleanupSelectChange(rowKeys: Array<string | number>) {
  cleanup.select(rowKeys);
}
function previousCleanupPage() {
  cleanup.previousPage();
}
function nextCleanupPage() {
  cleanup.nextPage();
}
async function startPull() {
  if (!pullReference.value.trim() || pulling.value) return;
  pulling.value = true;
  try {
    const receipt = await pullDockerImage({ reference: pullReference.value.trim() }, createPullIdempotencyKey());
    pullDrawerVisible.value = false;
    pullTaskId.value = receipt.task_id;
    pullTaskDrawerVisible.value = true;
    observePullTask(receipt.task_id);
    MessagePlugin.success(t('container.images.pull.accepted'));
  } catch {
    MessagePlugin.error(t('container.images.pull.failed'));
  } finally {
    pulling.value = false;
  }
}
function createPullIdempotencyKey() {
  const crypto = globalThis.crypto;
  const uuid = crypto?.randomUUID?.();
  if (uuid) return uuid;

  const entropy = new Uint32Array(4);
  crypto?.getRandomValues?.(entropy);
  // 不安全上下文可能不暴露 randomUUID；序号保证同一毫秒内的提交不会复用幂等键。
  pullIdempotencySequence += 1;
  return `container-image-pull-${Date.now()}-${pullIdempotencySequence}-${Array.from(entropy, (value) => value.toString(36)).join('')}`;
}
function resolveTaskType(taskType: string) {
  return taskType === CONTAINER_TASK_TYPE.DOCKER_IMAGE_PULL ? t('container.images.pull.taskType') : undefined;
}
function stopPullTaskObserver() {
  if (!pullTaskObserver) return;
  pullTaskObserver.stop();
  pullTaskObserver = null;
}
function observePullTask(taskId: number) {
  stopPullTaskObserver();
  pullTaskObserver = observeTask(taskId, {
    onError: (error) => logger.warn('docker image pull task observation failed', { error, taskId }),
    onTask: (task) => {
      if (!isTerminalTaskStatus(task.status)) return;
      stopPullTaskObserver();
      if (task.status === 'success') void refresh();
    },
  });
}
onUnmounted(() => {
  stopPullTaskObserver();
});
</script>
<style scoped lang="less">
.docker-images-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.docker-images-muted {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-images-identity {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.docker-images-identity__repository {
  color: var(--td-text-color-primary);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-images-card-list {
  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.docker-images-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-14);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.docker-images-card:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.docker-images-card--selected {
  border-color: var(--td-brand-color);
}

.docker-images-card__header,
.docker-images-card__primary,
.docker-images-card__actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.docker-images-card__identity {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.docker-images-card__identity strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  overflow-wrap: anywhere;
}

.docker-images-card__primary > strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.docker-images-card__details {
  display: grid;
  gap: var(--graft-density-gap-10);
  margin: 0;
}

.docker-images-card__details > div {
  align-items: baseline;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: minmax(72px, auto) minmax(0, 1fr);
  min-width: 0;
}

.docker-images-card__details dt,
.docker-images-card__details dd {
  margin: 0;
  min-width: 0;
}

.docker-images-card__details dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-images-card__details dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

.docker-images-card__actions > :first-child {
  flex: 0 0 auto;
}

.docker-images-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.docker-images-detail-loading-host {
  min-height: 240px;
}

.docker-images-detail-loading-host__indicator {
  display: grid;
  min-height: inherit;
  place-items: center;
}

@media (prefers-reduced-motion: no-preference) {
  .docker-images-detail-loading-host__indicator :deep(.t-icon-loading) {
    animation: docker-images-detail-loading-spin 1s linear infinite;
  }
}

@keyframes docker-images-detail-loading-spin {
  to {
    transform: rotate(360deg);
  }
}

.docker-images-detail-state {
  align-items: center;
  display: flex;
  justify-content: center;
  min-height: 240px;
  padding: var(--graft-density-gap-24) var(--graft-density-gap-16);
}

.docker-images-detail-error {
  padding: var(--graft-density-gap-16) 0;
}

.docker-images-detail__section {
  border-bottom: 1px solid var(--td-component-border);
  padding-bottom: var(--graft-density-gap-16);
}

.docker-images-detail__section:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.docker-images-detail__section h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0 0 var(--graft-density-gap-12);
}

.docker-images-detail__identity {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.docker-images-detail__identity strong {
  display: block;
  font: var(--td-font-title-medium);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-images-detail__label,
.docker-images-detail__grid dt,
.docker-images-detail__metadata dt {
  color: var(--td-text-color-secondary);
  display: block;
  font: var(--td-font-body-small);
  margin-bottom: var(--graft-density-gap-4);
}

.docker-images-detail__value {
  margin-top: var(--graft-density-gap-16);
}

.docker-images-detail__grid,
.docker-images-detail__metadata {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.docker-images-detail__metadata {
  grid-template-columns: 1fr;
}

.docker-images-detail__grid dd,
.docker-images-detail__metadata dd {
  color: var(--td-text-color-primary);
  margin: 0;
  overflow-wrap: anywhere;
}

.docker-images-detail__metadata dd .t-tooltip,
.docker-images-detail__metadata dd > .docker-images-code {
  display: block;
  max-width: 100%;
}

.docker-images-code {
  font-family: var(--td-font-family-medium);
  overflow-wrap: anywhere;
}

.docker-images-risk {
  color: var(--td-warning-color-7);
}

.docker-images-risk-list {
  color: var(--td-text-color-secondary);
  margin: 0 0 var(--graft-density-gap-12);
  padding-left: var(--graft-density-gap-20);
}

.docker-images-result-summary {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: var(--graft-density-gap-16);
}

.docker-images-result-summary > div {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-border);
  padding: var(--graft-density-gap-12);
}

.docker-images-result-summary span {
  color: var(--td-text-color-secondary);
  display: block;
  font: var(--td-font-body-small);
}

.docker-images-result-summary strong {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-large);
  margin-top: var(--graft-density-gap-4);
}

.docker-images-result__global-message {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.docker-images-failure-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  max-height: min(52vh, 420px);
  overflow: auto;
  padding-right: var(--graft-density-gap-4);
}

.docker-images-failure-group {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.docker-images-failure-group__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.docker-images-failure-group__header h3,
.docker-images-failure-group__header p {
  margin: 0;
}

.docker-images-failure-group__header h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.docker-images-failure-group__header p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

.docker-images-failure-item {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-12);
}

.docker-images-failure-item__header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  min-width: 0;
}

.docker-images-failure-item__identity {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.docker-images-failure-item__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-images-failure-item__manage-tags {
  flex: none;
  white-space: nowrap;
}

.docker-images-failure-item :deep(.t-collapse) {
  margin-top: var(--graft-density-gap-8);
}

.docker-images-failure-item :deep(.t-collapse-panel__header) {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-images-cleanup {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  max-height: calc(70vh - 120px);
  overflow: auto;
}

.docker-images-cleanup__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
}

.docker-images-cleanup__header h2,
.docker-images-cleanup__header p,
.docker-images-cleanup-section-head h3,
.docker-images-cleanup-section-head span {
  margin: 0;
}

.docker-images-cleanup__header h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
}

.docker-images-cleanup__header p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

.docker-images-cleanup__header-icon {
  align-items: center;
  background: var(--td-brand-color-1);
  border-radius: var(--td-radius-medium);
  color: var(--td-brand-color-7);
  display: flex;
  flex: 0 0 40px;
  height: 40px;
  justify-content: center;
  width: 40px;
}

.docker-images-cleanup__header-icon :deep(.t-icon) {
  font-size: var(--td-font-size-title-large);
}

.docker-images-cleanup-summary {
  background: var(--td-brand-color-1);
  border: 1px solid var(--td-brand-color-3);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
}

.docker-images-cleanup-summary__stats {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.docker-images-cleanup-summary__stats > div {
  min-width: 0;
}

.docker-images-cleanup-summary__label {
  color: var(--td-text-color-secondary);
  display: block;
  font: var(--td-font-body-small);
  margin-bottom: var(--graft-density-gap-4);
}

.docker-images-cleanup-summary__stats strong {
  color: var(--td-brand-color-7);
  font: var(--td-font-title-large);
  margin-right: var(--graft-density-gap-4);
}

.docker-images-cleanup-warning {
  margin: 0;
}

.docker-images-cleanup-preview {
  min-width: 0;
}

.docker-images-cleanup-section-head,
.docker-images-cleanup-footer {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.docker-images-cleanup-section-head h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.docker-images-cleanup-section-head span,
.docker-images-cleanup-footer span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.docker-images-cleanup-preview-body {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: minmax(0, 1fr) 32px;
}

.docker-images-cleanup-table {
  min-width: 0;
}

.docker-images-cleanup-image {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: 18px minmax(0, 1fr);
  min-width: 0;
  width: 100%;
}

.docker-images-cleanup-image__icon {
  color: var(--td-text-color-secondary);
}

.docker-images-cleanup-image__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-images-cleanup-pager {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex-direction: column;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  justify-content: center;
}

.docker-images-cleanup-pager :deep(.t-button) {
  color: var(--td-text-color-secondary);
}

.docker-images-cleanup-pager :deep(.t-button:not(.t-is-disabled):hover) {
  color: var(--td-brand-color-7);
}

.docker-images-cleanup-footer strong {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-medium);
  margin-top: var(--graft-density-gap-4);
}

:deep(.docker-images-cleanup-dialog .t-dialog__body) {
  overflow: hidden;
}

:deep(.docker-images-cleanup-dialog .t-dialog__footer) {
  padding-top: 0;
}

@media (width <= 768px) {
  .docker-images-detail__grid {
    grid-template-columns: 1fr;
  }

  .docker-images-cleanup-summary__stats {
    grid-template-columns: 1fr;
  }

  .docker-images-cleanup-section-head,
  .docker-images-cleanup-footer {
    align-items: flex-start;
    flex-direction: column;
  }

  .docker-images-cleanup-footer :deep(.t-space) {
    width: 100%;
  }

  .docker-images-cleanup-footer :deep(.t-button) {
    flex: 1;
  }
}
</style>
