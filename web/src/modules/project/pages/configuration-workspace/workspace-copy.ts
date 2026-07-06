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

type WorkspaceCopyKey = keyof WorkspaceCopy;
type WorkspaceCopyTranslate = (key: string) => string;

const workspaceCopyKeyMap: Record<WorkspaceCopyKey, string> = {
  cancelAction: 'project.configurationWorkspace.copy.cancelAction',
  continueWithDiskAction: 'project.configurationWorkspace.copy.continueWithDiskAction',
  deployAction: 'project.configurationWorkspace.copy.deployAction',
  deployContinueWithDiskAction: 'project.configurationWorkspace.copy.deployContinueWithDiskAction',
  deployDirtyBody: 'project.configurationWorkspace.copy.deployDirtyBody',
  deployDirtyTitle: 'project.configurationWorkspace.copy.deployDirtyTitle',
  diffAction: 'project.configurationWorkspace.copy.diffAction',
  diffViewerAriaLabel: 'project.configurationWorkspace.copy.diffViewerAriaLabel',
  dirtyCloseBody: 'project.configurationWorkspace.copy.dirtyCloseBody',
  dirtyCloseTitle: 'project.configurationWorkspace.copy.dirtyCloseTitle',
  dirtyProjectActionBody: 'project.configurationWorkspace.copy.dirtyProjectActionBody',
  dirtyProjectActionTitle: 'project.configurationWorkspace.copy.dirtyProjectActionTitle',
  discardAction: 'project.configurationWorkspace.copy.discardAction',
  editorAriaLabel: 'project.configurationWorkspace.copy.editorAriaLabel',
  envRedeployHint: 'project.configurationWorkspace.copy.envRedeployHint',
  feedbackHint: 'project.configurationWorkspace.copy.feedbackHint',
  feedbackTitle: 'project.configurationWorkspace.copy.feedbackTitle',
  fileTreeHint: 'project.configurationWorkspace.copy.fileTreeHint',
  fileTreeTitle: 'project.configurationWorkspace.copy.fileTreeTitle',
  filesEmpty: 'project.configurationWorkspace.copy.filesEmpty',
  hideHiddenAction: 'project.configurationWorkspace.copy.hideHiddenAction',
  loadingFile: 'project.configurationWorkspace.copy.loadingFile',
  readonlyHint: 'project.configurationWorkspace.copy.readonlyHint',
  reloadAction: 'project.configurationWorkspace.copy.reloadAction',
  reloadConfirmBody: 'project.configurationWorkspace.copy.reloadConfirmBody',
  reloadConfirmTitle: 'project.configurationWorkspace.copy.reloadConfirmTitle',
  saveAction: 'project.configurationWorkspace.copy.saveAction',
  saveAndContinueAction: 'project.configurationWorkspace.copy.saveAndContinueAction',
  saveFailed: 'project.configurationWorkspace.copy.saveFailed',
  saveSuccess: 'project.configurationWorkspace.copy.saveSuccess',
  saveThenContinueAction: 'project.configurationWorkspace.copy.saveThenContinueAction',
  savingAction: 'project.configurationWorkspace.copy.savingAction',
  selectDiffFile: 'project.configurationWorkspace.copy.selectDiffFile',
  selectFileToStart: 'project.configurationWorkspace.copy.selectFileToStart',
  showHiddenAction: 'project.configurationWorkspace.copy.showHiddenAction',
  snapshotAction: 'project.configurationWorkspace.copy.snapshotAction',
  snapshotDrawerTitle: 'project.configurationWorkspace.copy.snapshotDrawerTitle',
  snapshotViewerAriaLabel: 'project.configurationWorkspace.copy.snapshotViewerAriaLabel',
  summaryDescription: 'project.configurationWorkspace.copy.summaryDescription',
  summaryOpenTabsLabel: 'project.configurationWorkspace.copy.summaryOpenTabsLabel',
  summaryTitle: 'project.configurationWorkspace.copy.summaryTitle',
  summaryWorkingDirectoryLabel: 'project.configurationWorkspace.copy.summaryWorkingDirectoryLabel',
  tabsEmpty: 'project.configurationWorkspace.copy.tabsEmpty',
  validateAction: 'project.configurationWorkspace.copy.validateAction',
};

export function resolveConfigurationWorkspaceCopy(translate: WorkspaceCopyTranslate): WorkspaceCopy {
  return Object.fromEntries(
    Object.entries(workspaceCopyKeyMap).map(([field, key]) => [field, String(translate(key))]),
  ) as WorkspaceCopy;
}
