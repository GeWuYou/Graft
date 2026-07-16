type ApplicationCreated = {
  application_id: string;
};

export type OptionalDeploymentStatus = 'not-requested' | 'succeeded' | 'failed';

export type CreateWithOptionalDeployResult<T extends ApplicationCreated> = {
  created: T;
  deployment: {
    status: OptionalDeploymentStatus;
    error?: unknown;
  };
};

/**
 * 创建应用，并根据配置决定是否执行部署。
 *
 * @param options - 创建、部署及部署开关配置
 * @returns 包含已创建应用及部署状态的结果；部署失败时包含错误信息
 */
export async function createWithOptionalDeploy<T extends ApplicationCreated>(options: {
  create: () => Promise<T>;
  deploy: (applicationId: string) => Promise<unknown>;
  deployAfterCreate: boolean;
}): Promise<CreateWithOptionalDeployResult<T>> {
  const created = await options.create();
  if (!options.deployAfterCreate) {
    return { created, deployment: { status: 'not-requested' } };
  }

  try {
    await options.deploy(created.application_id);
    return { created, deployment: { status: 'succeeded' } };
  } catch (error) {
    return { created, deployment: { status: 'failed', error } };
  }
}
