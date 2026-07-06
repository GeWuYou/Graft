type WorkspaceCopy = {
  cancelAction: string;
  continueWithDiskAction: string;
  deployAction: string;
  deployContinueWithDiskAction: string;
  deployDirtyBody: string;
  deployDirtyTitle: string;
  diffAction: string;
  diffViewerAriaLabel: string;
  dirtyCloseBody: string;
  dirtyCloseTitle: string;
  dirtyProjectActionBody: string;
  dirtyProjectActionTitle: string;
  discardAction: string;
  editorAriaLabel: string;
  envRedeployHint: string;
  feedbackHint: string;
  feedbackTitle: string;
  fileTreeHint: string;
  fileTreeTitle: string;
  filesEmpty: string;
  hideHiddenAction: string;
  loadingFile: string;
  readonlyHint: string;
  reloadAction: string;
  reloadConfirmBody: string;
  reloadConfirmTitle: string;
  saveAction: string;
  saveAndContinueAction: string;
  saveFailed: string;
  saveSuccess: string;
  saveThenContinueAction: string;
  savingAction: string;
  selectDiffFile: string;
  selectFileToStart: string;
  showHiddenAction: string;
  snapshotAction: string;
  snapshotDrawerTitle: string;
  snapshotViewerAriaLabel: string;
  summaryDescription: string;
  summaryOpenTabsLabel: string;
  summaryTitle: string;
  summaryWorkingDirectoryLabel: string;
  tabsEmpty: string;
  validateAction: string;
};

export function resolveConfigurationWorkspaceCopy(locale: string): WorkspaceCopy {
  const isZhCn = locale.toLowerCase().startsWith('zh');
  if (isZhCn) {
    return {
      cancelAction: '取消',
      continueWithDiskAction: '继续使用磁盘版本',
      deployAction: '部署项目',
      deployContinueWithDiskAction: '继续使用磁盘版本部署',
      deployDirtyBody: '检测到未保存的修改，是否先保存？',
      deployDirtyTitle: '未保存修改',
      diffAction: '执行差异比较',
      diffViewerAriaLabel: '项目配置差异查看器',
      dirtyCloseBody: '当前文件有未保存的修改，关闭前请选择处理方式。',
      dirtyCloseTitle: '关闭前保存修改',
      dirtyProjectActionBody: '检测到未保存的修改，是否先保存后继续当前项目级操作？',
      dirtyProjectActionTitle: '存在未保存修改',
      discardAction: '放弃',
      editorAriaLabel: '项目文件编辑器',
      envRedeployHint: '修改环境变量后通常需要重新部署项目才能生效。',
      feedbackHint: '校验和差异结果会在这里保留，不会覆盖文件编辑上下文。',
      feedbackTitle: '反馈工作区',
      fileTreeHint: '按项目根目录加载真实文件结构，仅展示接口返回的目录与文件。',
      fileTreeTitle: 'Project Files',
      filesEmpty: '当前目录没有可浏览的文件。',
      hideHiddenAction: '隐藏默认目录',
      loadingFile: '正在加载文件内容…',
      readonlyHint: '当前文件只读，可查看但不能保存。',
      reloadAction: '重新加载',
      reloadConfirmBody: '重新加载会丢弃当前文件未保存的修改。',
      reloadConfirmTitle: '重新加载文件',
      saveAction: '保存',
      saveAndContinueAction: '保存并继续',
      saveFailed: '文件保存失败。',
      saveSuccess: '文件已保存到工作目录。',
      saveThenContinueAction: '保存',
      savingAction: '保存中',
      selectDiffFile: '选择一个文件查看差异详情',
      selectFileToStart: '选择左侧文件开始浏览或编辑。',
      showHiddenAction: '显示默认隐藏目录',
      snapshotAction: '查看当前快照',
      snapshotDrawerTitle: '当前快照',
      snapshotViewerAriaLabel: '项目当前快照查看器',
      summaryDescription: '工作台以项目根目录为准，支持文件浏览、编辑、保存，以及基于磁盘状态执行差异、校验和部署。',
      summaryOpenTabsLabel: '已打开标签',
      summaryTitle: '工作台摘要',
      summaryWorkingDirectoryLabel: '工作目录',
      tabsEmpty: '当前还没有打开文件。',
      validateAction: '执行校验',
    };
  }

  return {
    cancelAction: 'Cancel',
    continueWithDiskAction: 'Continue with Disk Version',
    deployAction: 'Deploy Project',
    deployContinueWithDiskAction: 'Deploy from Saved Disk State',
    deployDirtyBody: 'Unsaved changes were detected. Save them before deploy?',
    deployDirtyTitle: 'Unsaved Changes',
    diffAction: 'Run Diff',
    diffViewerAriaLabel: 'Project Configuration Diff Viewer',
    dirtyCloseBody: 'This file has unsaved changes. Choose how to proceed before closing it.',
    dirtyCloseTitle: 'Save Changes Before Closing',
    dirtyProjectActionBody: 'Unsaved changes were detected. Save them before continuing this project action?',
    dirtyProjectActionTitle: 'Unsaved Changes Detected',
    discardAction: 'Discard',
    editorAriaLabel: 'Project File Editor',
    envRedeployHint: 'Environment variable changes usually require a project redeploy before they take effect.',
    feedbackHint: 'Validation and diff results stay here without disrupting the current file buffer.',
    feedbackTitle: 'Feedback Workspace',
    fileTreeHint: 'Load the real project-root structure from the API. No filenames are guessed on the client.',
    fileTreeTitle: 'Project Files',
    filesEmpty: 'No files are available in this directory.',
    hideHiddenAction: 'Hide Default Hidden Directories',
    loadingFile: 'Loading file content…',
    readonlyHint: 'This file is read-only. You can inspect it, but you cannot save it.',
    reloadAction: 'Reload File',
    reloadConfirmBody: 'Reloading will discard the current unsaved edits for this file.',
    reloadConfirmTitle: 'Reload File',
    saveAction: 'Save',
    saveAndContinueAction: 'Save and Continue',
    saveFailed: 'Failed to save the file.',
    saveSuccess: 'The file was saved to the working directory.',
    saveThenContinueAction: 'Save',
    savingAction: 'Saving',
    selectDiffFile: 'Choose a file to inspect its diff',
    selectFileToStart: 'Select a file from the left tree to browse or edit it.',
    showHiddenAction: 'Show Default Hidden Directories',
    snapshotAction: 'View Current Snapshot',
    snapshotDrawerTitle: 'Current Snapshot',
    snapshotViewerAriaLabel: 'Current Project Snapshot Viewer',
    summaryDescription:
      'Use the project-root workbench to browse files, edit content, save to disk, and run diff, validation, or deploy against saved state.',
    summaryOpenTabsLabel: 'Open Tabs',
    summaryTitle: 'Workspace Summary',
    summaryWorkingDirectoryLabel: 'Working Directory',
    tabsEmpty: 'No file is open yet.',
    validateAction: 'Run Validate',
  };
}
