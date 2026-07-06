type WorkspaceCopy = {
  composeEditorAriaLabel: string;
  diffViewerAriaLabel: string;
  draftsHint: string;
  draftsTitle: string;
  envEditorAriaLabel: string;
  feedbackHint: string;
  feedbackTitle: string;
  noSourceFiles: string;
  normalizedPreviewAriaLabel: string;
  snapshotViewerAriaLabel: string;
  selectDiffFile: string;
  sourceViewerAriaLabel: string;
  snapshotAction: string;
  snapshotDrawerTitle: string;
  sourceAction: string;
  sourceDrawerTitle: string;
  summaryDescription: string;
  summaryTitle: string;
};

// TODO: move these labels into `web/src/modules/project/locales/**` when the routing/locale worker wires the page.
export function resolveConfigurationWorkspaceCopy(locale: string): WorkspaceCopy {
  const isZhCn = locale.toLowerCase().startsWith('zh');
  if (isZhCn) {
    return {
      composeEditorAriaLabel: 'Compose 草稿编辑器',
      diffViewerAriaLabel: '配置草稿差异查看器',
      draftsHint: '选择要编辑的项目文件。',
      draftsTitle: '工程文件',
      envEditorAriaLabel: '环境变量草稿编辑器',
      feedbackHint: '校验与差异结果只在这个反馈面板里展开，不再平铺占满首屏。',
      feedbackTitle: '反馈工作区',
      noSourceFiles: '当前没有可查看的源文件。',
      normalizedPreviewAriaLabel: '规范化 Compose 预览',
      snapshotViewerAriaLabel: '当前快照查看器',
      selectDiffFile: '选择要查看的差异文件',
      sourceViewerAriaLabel: '源文件查看器',
      snapshotAction: '查看当前快照',
      snapshotDrawerTitle: '当前快照',
      sourceAction: '查看源文件',
      sourceDrawerTitle: '源文件',
      summaryDescription: '把当前配置状态、草稿编辑、差异校验和部署操作收敛到一个连续工作流里。',
      summaryTitle: '配置摘要',
    };
  }

  return {
    composeEditorAriaLabel: 'Compose Draft Editor',
    diffViewerAriaLabel: 'Configuration Draft Diff Viewer',
    draftsHint: 'Select the project file you want to edit.',
    draftsTitle: 'Project Files',
    envEditorAriaLabel: 'Env Draft Editor',
    feedbackHint: 'Validation and diff results stay in this workspace instead of competing for first-screen space.',
    feedbackTitle: 'Feedback Workspace',
    noSourceFiles: 'No source files are available for this project.',
    normalizedPreviewAriaLabel: 'Normalized Compose Preview',
    snapshotViewerAriaLabel: 'Current Snapshot Viewer',
    selectDiffFile: 'Choose a file to inspect its diff',
    sourceViewerAriaLabel: 'Source File Viewer',
    snapshotAction: 'View Current Snapshot',
    snapshotDrawerTitle: 'Current Snapshot',
    sourceAction: 'View Source Files',
    sourceDrawerTitle: 'Source Files',
    summaryDescription:
      'Keep configuration status, draft editing, diff checks, and deployment in one continuous workflow.',
    summaryTitle: 'Configuration Summary',
  };
}
