import type { CreateUpdateOperationRequest, UpdateOperation, UpdateStatus } from './update';

/** 开发预览页通过此边界替换真实更新 API，避免本地 UI 验收触发任何更新操作。 */
export type UpdateCenterDataSource = {
  permissions: {
    check: boolean;
    manage: boolean;
  };
  getStatus(): Promise<UpdateStatus>;
  checkForUpdates(): Promise<UpdateStatus>;
  getOperations(): Promise<UpdateOperation[]>;
  createOperation(payload: CreateUpdateOperationRequest): Promise<UpdateOperation>;
};
